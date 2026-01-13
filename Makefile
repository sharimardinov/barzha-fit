APP_NAME=fitness-bot
CMD_PATH=./cmd/bot/main.go
BIN_DIR=bin

.PHONY: run build clean dev

run:
	go run $(CMD_PATH)

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(CMD_PATH)

dev:
	go run $(CMD_PATH)

clean:
	rm -rf $(BIN_DIR)

