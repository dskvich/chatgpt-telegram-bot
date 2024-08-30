package domain

type Callback string

const (
	DrawCallback       = "draw"
	SetChatTTLCallback = "ttl_"
	SettingsCallback   = "edit_settings"
)
