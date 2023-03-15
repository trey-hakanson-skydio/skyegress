package config

type Config struct {
	LiveKitConfig livekitConfig `kong:"embed"`
}

type livekitConfig struct {
	Host      string `kong:"required,help='LiveKit host',env=LIVEKIT_URL"`
	ApiKey    string `kong:"required,help='LiveKit server API key',env=LIVEKIT_API_KEY"`
	ApiSecret string `kong:"required,help='LiveKit server API secret',env=LIVEKIT_API_SECRET"`
}
