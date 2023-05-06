### Chat GPT proxy telegram bot
## Cases
Post a message to a bot in direct chat.
Telegram bot proxies request to Chat GPT `gpt-3.5-turbo` model.
Once chat GPT replies, telegram bot also replies to the message sender.

## Usage
```
go install github.com/antelman107/chatgpt-telegram-bot

TELEGRAM_BOT_TOKEN=xxx GPT_TOKEN=xxx chatgpt-telegram-bot
// post message to the bot
2023/05/06 11:53:48 [User] Message
```
