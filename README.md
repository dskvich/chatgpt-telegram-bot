## Chat GPT proxy Telegram bot
This repository contains code for a Telegram bot that acts as a proxy between users and the Chat GPT gpt-3.5-turbo model.
Users can send messages to the bot, which then forwards the requests to the Chat GPT model for processing.
The generated replies are sent back to the users via the bot.

### Development
After clone set the `TELEGRAM_BOT_TOKEN` and `GPT_TOKEN` environment variables.

To start the server:
```
go run main.go
```
Then post a message to the bot.

`TELEGRAM_AUTHORIZED_USER_IDS` environment variable can be used to restrict the access to the bot, allowing only authorized users to utilize its features.
Telegram user IDs should be provided as space-separated values within the environment variable. 
If the `TELEGRAM_AUTHORIZED_USER_IDS` variable is empty, all users will be permitted to use the bot by default.

### How to Create a New Bot for Telegram
- Enter @Botfather in the search tab and choose this bot.
- Choose or type the /newbot command and send it.
- Enter a name for the bot and give it a unique username. Note that the bot name must end with the word "bot" (case-insensitive).
- Copy and save the Telegram bot's access token for later steps.