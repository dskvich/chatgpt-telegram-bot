set GOOS=linux
go build -o main main.go
build-lambda-zip.exe -o main.zip main
del main