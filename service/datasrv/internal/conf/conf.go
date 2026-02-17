package conf

var (
	Conf = new(Config)
)

type Config struct {
	Github GithubConfig
}

type GithubConfig struct {
	Token string
}

func (Config) Print() {}
