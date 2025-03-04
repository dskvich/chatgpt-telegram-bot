package domain

type Response struct {
	Text  string
	Image *Image
	File  *File
	Err   error
}

type Image struct {
	ID   int64
	Data []byte
}

type File struct {
	Name string
	Data []byte
}
