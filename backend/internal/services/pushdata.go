package services

import (
	"encoding/json"
	"strings"
)

// 第三方推送字段别名（不同渠道命名不一致，统一做宽松匹配）
var pushFieldAliases = map[string][]string{
	"phone": {
		"phone", "mobile", "tel", "telephone", "phonenumber", "contactphone",
		"号码", "手机号", "手机", "联系电话", "customerphone",
	},
	"condition": {
		"condition", "illness", "disease", "conditiontext", "symptom", "clueCondition",
		"病情", "病症", "意向", "项目",
	},
	"doctor": {
		"doctor", "doctorname", "attendingdoctor", "doctor_name",
		"医生", "接诊医生", "主治医生",
	},
	"name": {
		"name", "customername", "contactname", "username",
		"姓名", "客户姓名", "联系人",
	},
}

// ExtractPushExtra 从推送原始 JSON 中提取补充字段（宽松别名匹配，忽略大小写）
// 返回 key 为 phone / condition / doctor / name，值为去除首尾空白后的字符串
func ExtractPushExtra(rawJSON string) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(rawJSON) == "" {
		return out
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &m); err != nil {
		return out
	}
	// 转成小写 key 索引（保留首次出现的值）
	lower := make(map[string]string, len(m))
	for k, v := range m {
		lk := strings.ToLower(strings.TrimSpace(k))
		if _, exists := lower[lk]; exists {
			continue
		}
		lower[lk] = stringifyValue(v)
	}
	for field, aliases := range pushFieldAliases {
		for _, a := range aliases {
			if v, ok := lower[strings.ToLower(a)]; ok && strings.TrimSpace(v) != "" {
				out[field] = strings.TrimSpace(v)
				break
			}
		}
	}
	return out
}

// PickString 从解析后的 JSON 对象中按 key 取字符串值（数字/布尔自动转文本，手机号不会被科学计数法破坏）
func PickString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		return strings.TrimSpace(stringifyValue(v))
	}
	return ""
}

func stringifyValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case float64:
		// 避免手机号被解析成 1.38e+10
		if val == float64(int64(val)) {
			return json.Number(strings.TrimSpace(formatInt(int64(val)))).String()
		}
		return strings.TrimSpace(formatFloat(val))
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func formatFloat(f float64) string {
	b, err := json.Marshal(f)
	if err != nil {
		return ""
	}
	return string(b)
}

// BuildCopyText 生成「序号 → 资质所属地」这段可直接粘贴的纯文本
func BuildCopyText(d *MailData) string {
	lines := []string{
		"【序号】" + d.Sequence,
		"【渠道】" + d.Channel,
		"【号码】" + d.Phone,
		"【病情】" + d.Condition,
		"【运营】" + d.Nickname,
		"【医生】" + d.Doctor,
		"【资质所属地】" + d.QualificationAddress,
	}
	return strings.Join(lines, "\n")
}
