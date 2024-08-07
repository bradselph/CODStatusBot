# COD Status Bot

[![Go](https://github.com/bradselph/CODStatusBot/actions/workflows/go.yml/badge.svg)](https://github.com/bradselph/CODStatusBot/actions/workflows/go.yml)

## Introduction

COD Status Bot is a Discord bot designed to help you monitor your Activision accounts for shadowbans or permanent bans in Call of Duty games. The bot periodically checks the status of your accounts and notifies you of any changes.

## Features

- Monitor multiple Activision accounts
- Periodic automatic checks (customizable interval with your own API key)
- Manual status checks
- Account age verification
- Ban history logs
- Customizable notification preferences
- Anonymous feedback submission
- EZ-Captcha integration for improved reliability

## Getting Started

1. Invite the bot to your Discord server using the provided [Invite Link](https://discord.com/oauth2/authorize?client_id=1211857854324015124).
2. Once the bot joins your server, it will automatically register the necessary commands.
3. (Optional) Get your own EZ-Captcha API key for customized check intervals and improved service.

## EZ-Captcha Integration

The bot now uses EZ-Captcha for solving CAPTCHAs, which improves the reliability of account status checks. Users have two options:

1. Use the bot's default API key (limited use, shared among all users)
2. Get your own EZ-Captcha API key for unlimited use and customizable check intervals

### Getting Your Own EZ-Captcha API Key

1. Visit [EZ-Captcha Registration](https://dashboard.ez-captcha.com/#/register?inviteCode=uyNrRgWlEKy)
2. Complete the registration process
3. Once registered, you'll receive your API key
4. Use the `/setcaptchaservice` command to set your API key in the bot

By using your own API key, you can customize the check interval for your accounts and enjoy unlimited use of the service.

## Commands

### /addaccount

Add a new account to be monitored by the bot.

Usage: `/addaccount`

This command will open a modal where you can enter:
- Account Title: A name to identify the account
- SSO Cookie: The Single Sign-On cookie associated with your Activision account
- EZ-Captcha API Key (optional): Your personal API key for unlimited use

### /removeaccount

Remove an account from being monitored by the bot.

Usage: `/removeaccount`

This command will display a list of your monitored accounts and prompt you to confirm the removal of the selected account. All related data will be permanently deleted from the bot.

### /updateaccount

Update the SSO cookie or EZ-Captcha API key for an existing account.

Usage: `/updateaccount`

This command will display a list of your monitored accounts and allow you to update the SSO cookie or EZ-Captcha API key for the selected account.

### /listaccounts

List all your monitored accounts.

Usage: `/listaccounts`

### /accountlogs

View the status change logs for a specific account.

Usage: `/accountlogs`

This command will display a list of your monitored accounts and show the logs for the selected account.

### /accountage

Check the age of a specific account.

Usage: `/accountage`

This command will display a list of your monitored accounts and show the age of the selected account.

### /checknow

Immediately check the status of all your accounts or a specific account.

Usage: `/checknow [account_title]`

- `account_title` (optional): The title of the specific account to check. If omitted, all accounts will be checked.

### /setpreference

Set your preference for where you want to receive status notifications.

Usage: `/setpreference <type>`

- `type`: Choose between "channel" (default) or "dm" for direct messages.

### /setcaptchaservice

Set your personal EZ-Captcha API key for unlimited use and customizable check intervals.

Usage: `/setcaptchaservice <api_key>`

- `api_key`: Your personal EZ-Captcha API key. Leave blank to use the bot's default key.

### /help

Display a guide on how to obtain your SSO cookie.

Usage: `/help`

### /feedback

Send anonymous feedback or suggestions to the bot developer.

Usage: `/feedback <message>`

- `message`: Your feedback or suggestion.

## Notifications

The bot will send notifications:

- When there's a change in the ban status of an account
- Daily for each account, confirming that it's still being monitored
- If an SSO cookie becomes invalid or expires

Notifications will be sent to the channel where the account was added or to your DMs, depending on your preference set with `/setpreference`.

## SSO Cookie

To get the SSO cookie:

1. Log in to your Activision account on a web browser.
2. Open the browser's developer tools.
3. Navigate to the Application tab, then Cookies.
4. Find the cookie named `ACT_SSO_COOKIE` associated with the Activision domain.

**Important:** Keep your SSO cookie confidential and do not share it with anyone.

## Support

If you encounter any issues or have questions, please contact the bot developer through Discord or the platform where you discovered this bot.

## Note on Data Privacy

The bot stores minimal data necessary for its operation, including account titles, SSO cookies, and status logs. This data is used solely for the purpose of monitoring account status and providing notifications. The bot does not share this data with any third parties. Users can use the provided commands to delete their data from the bot at any time, ensuring that no data is left afterwards.

## Disclaimer

This bot is not affiliated with or endorsed by Activision. Use it at your own risk. The developers are not responsible for any consequences resulting from the use of this bot.