package config

type Config struct {
	LiveKitConfig LiveKitConfig `kong:"embed"`
}

type HTTPConfig struct {
	Port int `kong:"default=8008"`
}

type RTSPConfig struct {
	Port              int    `kong:"default=8554"`
	UDPRTPPort        int    `kong:"default=8000"`
	UDPRTCPPort       int    `kong:"default=8001"`
	MulticastIPRange  string `kong:"default='224.1.0.0/16'"`
	MulticastRTPPort  int    `kong:"default=8002"`
	MulticastRTCPPort int    `kong:"default=8003"`
}

type LiveKitConfig struct {
	Host      string `kong:"required,help='LiveKit host',env=LIVEKIT_URL"`
	ApiKey    string `kong:"required,help='LiveKit server API key',env=LIVEKIT_API_KEY"`
	ApiSecret string `kong:"required,help='LiveKit server API secret',env=LIVEKIT_API_SECRET"`
}
