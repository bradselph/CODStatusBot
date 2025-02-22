# COD Status Bot

[![Go](https://github.com/bradselph/CODStatusBot/actions/workflows/go.yml/badge.svg)](https://github.com/bradselph/CODStatusBot/actions/workflows/go.yml)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fbradselph%2FCODStatusBot.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fbradselph%2FCODStatusBot?ref=badge_shield)

## Introduction

COD Status Bot is a Discord bot designed to help you monitor your Activision accounts for shadowbans or permanent bans
in Call of Duty games. The bot periodically checks the status of your accounts and notifies you of any changes. Now
serving 395+ Discord servers, our bot has been optimized for performance and scalability.

## Features

- Monitor multiple Activision accounts
- Periodic automatic checks (customizable interval with your own API key)
- Manual status checks
- Account age verification
- Ban history logs
- Customizable notification preferences
- Anonymous feedback submission
- EZ-Captcha and 2Captcha integration for continued compatibility with Activision
- SSO Cookie expiration tracking and notifications
- Toggle automatic checks on/off for individual accounts
- VIP account status detection and tracking

## Getting Started

1. Invite the bot to your Discord server using the
   provided [Invite Link](https://discord.com/oauth2/authorize?client_id=1211857854324015124).
2. Once the bot joins your server, it will automatically register the necessary commands.
3. Set up your EZ-Captcha or 2Captcha API key for full functionality and customized check intervals.
4. Use the `/addaccount` command to start monitoring your first account.

## Captcha Service Integration

The bot uses EZ-Captcha for solving CAPTCHAs, which maintains compatibility to check your accounts with
Activision. Users have two options:

1. Use the bot's default API key (limited use, shared among all users)
2. Get your own API key for unlimited use and customizable check intervals

### Getting Your Own API Key

#### EZ-Captcha:
1. Visit [EZ-Captcha Registration](https://dashboard.ez-captcha.com/#/register?inviteCode=uyNrRgWlEKy)
2. Complete the registration process
3. Once registered, you'll receive your API key
4. Use the `/setcaptchaservice` command to set your API key in the bot

#### 2Captcha:
1. Visit [2Captcha](https://2captcha.com/) to register
2. Purchase credits for your account
3. Obtain your API key from your dashboard

Use the `/setcaptchaservice` command to set your preferred service and API key in the bot.

## Commands

### Account Management Commands

### /addaccount
Add a new account to be monitored by the bot. You'll need to provide:
- Account Title: A name to identify the account
- SSO Cookie: The Single Sign-On cookie associated with your Activision account

### /removeaccount
Remove an account from being monitored by the bot. This will delete all associated data.

### /updateaccount
Update the SSO cookie for an existing account. Use this when your cookie expires or becomes invalid.

### /listaccounts
List all your monitored accounts, including:
- Current status
- VIP status
- Cookie expiration
- Check status (enabled/disabled)
- Notification preferences

### /accountlogs
View the status change logs for a specific account or all accounts. This shows the last 10 status changes.

### /accountage
Check the age and VIP status of a specific account. Displays:
- Account creation date
- Current age
- VIP status
- Last status check

Check the age of a specific account. This displays the account's creation date and current age.
### Status Check Commands

### /checknow
Immediately check the status of all your accounts or a specific account. Note: This command is rate-limited for users without a personal API key.

Immediately check the status of all your accounts or a specific account. This command is rate-limited for users without
### /togglecheck
Toggle automatic checks on/off for a monitored account. Useful for:
- Temporarily disabling checks on specific accounts
- Re-enabling previously disabled accounts
- Managing account monitoring status

### Configuration Commands

### /setcheckinterval
Configure monitoring preferences:
- Check Interval: How often the bot checks your accounts (1-1440 minutes)
- Notification Interval: How often you receive status updates (1-24 hours)
- Cooldown Duration: Minimum time between repeated notifications (1-24 hours)
- Status Change Cooldown: Minimum time between status change notifications (1-24 hours)

### /setnotifications
Set your notification preferences:
- Channel: Receive notifications in the Discord channel
- DM: Receive notifications via direct message

### /setcaptchaservice
Configure your captcha service:
- Select provider (EZ-Captcha or 2Captcha)
- Set your personal API key
- Check current balance

Set your personal EZ-Captcha API key for unlimited use and customizable check intervals.
### /checkcaptchabalance
Check your remaining balance with your configured captcha service.

### Help and Support Commands

### /helpapi
Display a detailed guide on:
- Using the bot effectively
- Setting up your API key
- Understanding available commands

### /helpcookie
Comprehensive guide on obtaining your SSO cookie, including:
- Step-by-step instructions
- Browser-specific methods
- Troubleshooting tips

### /feedback
Send anonymous feedback or suggestions to the bot developer. Features:
- Optional anonymity
- Direct delivery to developer
- Support for feature requests and bug reports

## Notifications

The bot will send notifications for:
- Ban status changes (permanent, temporary, or shadowban)
- Daily account monitoring confirmations
- Invalid or expiring SSO cookies
- Cookie expiration warnings (24 hours notice)
- Captcha service balance warnings
- VIP status changes

Notifications can be sent to:
- The channel where the account was added
- Your DMs (configurable via `/setnotifications`)

## SSO Cookie

The SSO (Single Sign-On) cookie is required to authenticate with Activision's services. To get the SSO cookie:

1. Log in to your Activision account on a web browser.
2. Open the browser's developer tools (usually F12 or right-click and select "Inspect").
3. Navigate to the Application or Storage tab.
4. Find the cookie named `ACT_SSO_COOKIE` associated with the Activision domain.
5. Copy the entire value of this cookie.

For a detailed guide, use the `/helpcookie` command.

## Rate Limiting

To prevent abuse and ensure fair usage:

- Users without a personal API key are subject to rate limits on the `/checknow` command.
- Global cooldowns are implemented for notifications to prevent spam.

## Database and Data Management

The bot uses a MySQL database to store account information and user settings. It includes:

- Secure storage of SSO cookies and user preferences
- Regular checks for expired cookies and account status changes
- Optimized queries and connection pooling for high-performance

## Support and Feedback

If you encounter any issues or have questions:

1. Use the `/feedback` command to contact the bot developer anonymously.

## Privacy and Data Security

- The bot stores minimal data necessary for operation: account titles, SSO cookies, and status logs.
- Data is used solely for monitoring account status and providing notifications.
- No data is shared with third parties.
- Users can delete their data at any time using the `/removeaccount` command.
- We employ industry-standard security practices to protect your data.

## Recent Changes and Updates
- Optimized bot performance for 395+ Discord servers
- Enhanced rate limiting to ensure fair usage across all servers
- Improved error handling and logging for better issue resolution
- Added support for multiple captcha services (EZ-Captcha and 2Captcha) Note: 2Captcha is not stable and may be disabled for use.
- 




## Disclaimer

This bot is not affiliated with or endorsed by Activision. Use it at your own risk. The developers are not responsible
for any consequences resulting from the use of this bot.

## Contributing

We welcome contributions to the COD Status Bot! If you'd like to contribute:

1. Fork the repository on GitHub.
2. Create a new branch for your feature or bug fix.
3. Commit your changes with clear, descriptive commit messages.
4. Push your branch and submit a pull request.

Please ensure your code adheres to the existing style and passes all tests. For major changes, please open an issue
first to discuss what you would like to change.

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0). This means:

- You can use, modify, and distribute this software freely.
- If you modify the software and use it to provide a service over a network, you must make your modified source code
  available to users of that service.
- Any modifications or larger works must also be licensed under AGPL-3.0.

For more details, see the [LICENSE](LICENSE) file in the repository or
visit [GNU AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0.en.html).


[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fbradselph%2FCODStatusBot.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fbradselph%2FCODStatusBot?ref=badge_large)

## Open Source

This project is open source and available on GitHub. We believe in the power of community-driven development and welcome
contributions from developers around the world. By making this bot open source, we aim to:

- Encourage collaboration and improvement of the bot's features.
- Provide transparency in how the bot operates.
- Enable the community to adapt the bot for their specific needs.

You can find the full source code, contribute to the project, or report issues at
our [GitHub repository](https://github.com/bradselph/CODStatusBot).

Thank you for using COD Status Bot! We're committed to providing a reliable and efficient service to our growing
community of users across 300+ Discord servers.