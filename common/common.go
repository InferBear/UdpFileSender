package common

// 文件请求结构
type FileRequest struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// 文件响应结构
type FileResponse struct {
	Content []byte `json:"content"`
	Start   int64  `json:"start"`
	End     int64  `json:"end"`
	MD5Hash string `json:"md5hash"`
}
