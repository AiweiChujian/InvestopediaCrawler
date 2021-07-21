package investopediaCrawler

import "errors"

var (
	ERR_DETAIL_NO_PARSEFUNC = errors.New("详情页没有对应解析方法")
	ERR_CANNOT_PARSE_UPDATED = errors.New("未知的日期显示格式")
)
