all:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm                                 \
	CC=/usr/local/arm-linux-gcc-4.4.3/bin/arm-none-linux-gnueabi-gcc 	\
	go build -ldflags 													\																\
	-o lcrtu-update main.go
