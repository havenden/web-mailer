package services

import "errors"

var errRecordNotRetryable = errors.New("该邮件记录当前状态不支持重发（仅「发送失败」或「重试等待」可重发）")
