package constants

type ProviderType string

const (
	GoogleProvider   ProviderType = "google"
	GithubProvider   ProviderType = "github"
	FacebookProvider ProviderType = "facebook"
	LocalProvider    ProviderType = "local"
)

func (p ProviderType) String() string {
	return string(p)
}
