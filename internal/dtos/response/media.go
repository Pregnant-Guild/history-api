package response

type PreSignedResponse struct {
	UploadUrl     string            `json:"uploadUrl"`
	PublicUrl     string            `json:"publicUrl"`
	FileName      string            `json:"fileName"`
	MediaId       string            `json:"mediaId"`
	SignedHeaders map[string]string `json:"signedHeaders"`
}
