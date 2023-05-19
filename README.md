### Chat GPT proxy telegram bot
## Cases
Post a message to a bot in direct chat.
Telegram bot proxies request to Chat GPT `gpt-3.5-turbo` model.
Once chat GPT replies, telegram bot also replies to the message sender.

## Usage
```bash
TBD
```

## Configuration

### Lambda function creation
* Go to Lambda's web page: https://console.aws.amazon.com/lambda
* Press Create function on the top right.
* Choose Author from scratch.
* Name your function and choose the Go 1.x runtime.
* In the execution roles, choose Use an existing role and choose the one we created earlier.
* Once the function has been created, write main as the function handler.

### API Gateway configuration
* Go to the API Gateway's web page: https://console.aws.amazon.com/apigateway
* Go to API and choose Create API.
* Choose New API and use a Regional endpoint.
* Click on the newly created API and, from the dropdown Actions menu, choose Create Method.
* Choose the POST method and confirm by pressing on the tick.
* Make sure that Lambda function is selected as the Integration type.
* Make sure that Lambda Proxy Integration is disabled.
* Choose the appropriate region and write name of the function you've created in the Lambda function field.
* Make sure that in the Body mapping templates of the function, When there are no templates defined (recommended)" is selected.
* Deploy the API by choosing the option from the dropdown menu. This way you'll be given the URL we'll use to set up the bot's webhooks.