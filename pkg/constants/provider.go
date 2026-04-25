package constants

type ProviderType string

const (
	ProviderTypeGoogle   ProviderType = "google"
	ProviderTypeGithub   ProviderType = "github"
	ProviderTypeFacebook ProviderType = "facebook"
	ProviderTypeLocal    ProviderType = "local"
)

func (p ProviderType) String() string {
	return string(p)
}
