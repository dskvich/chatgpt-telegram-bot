package domain

type Response struct {
	ChatID   int64
	Text     string
	Image    *Image
	File     *File
	Keyboard *Keyboard
	Err      error
}

type Image struct {
	PromptID int64
	Data     []byte
}

type File struct {
	Name string
	Data []byte
}

type Keyboard struct {
	Title          string
	ButtonLabels   []string
	CallbackPrefix string
	ButtonsPerRow  int
}
