package helpapi

import (
	"github.com/bradselph/CODStatusBot/logger"

	"github.com/bwmarrin/discordgo"
)

func CommandHelpApi(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger.Log.Info("Received help command")
	helpApiGuide := []string{
		"CODStatusBot Help Guide\n\n" +
			"To add your Call of Duty account to the bot, you'll need to obtain your SSO (Single Sign-On) cookie. Follow these steps:\n\n" +
			"1. **Login to Your Activision Account:**\n" +
			"   - Go to [Activision's website](https://www.activision.com/) and log in with the account you want to track.\n\n" +
			"2. **Access the Developer Console:**\n" +
			"   - Depending on your browser:\n" +
			"     - You can Press `F12` to open the developer console or right-click on the page and select \"Inspect\".\n\n" +
			"3. **Retrieve the SSO Cookie:**\n" +
			"   - In the developer console, switch to the \"Console\" tab.\n" +
			"   - Paste the following JavaScript code snippet:\n" +
			"```javascript\n" +
			"var cookieValue = document.cookie.match(/ACT_SSO_COOKIE=([^;]+)/)[1];\n" +
			"console.log(cookieValue);\n" +
			"```\n" +
			"   - Accept any warnings related to pasting code.\n\n" +
			"4. **Copy the Cookie Value:**\n" +
			"   - After executing the code, you'll see the SSO cookie value. Copy it.\n\n" +
			"5. **Add Your Account to the Bot:**\n" +
			"   - Continue by adding your account to the bot using the copied cookie value.\n\n",

		"## Additional Methods (Browser-Specific):\n" +
			"- **Firefox Users:**\n" +
			"  - Go to the \"Storage\" tab in the developer console.\n" +
			"  - Click on \"Cookies,\" then find the \"activision.com\" domain.\n" +
			"  - Locate the cookie named \"ACT_SSO_COOKIE\" and copy its value.\n\n" +
			"- **Chrome Users:**\n" +
			"  - Navigate to the \"Application\" tab in the developer console.\n" +
			"  - Click on \"Cookies,\" then find the \"activision.com\" domain.\n" +
			"  - Look for the cookie named \"ACT_SSO_COOKIE\" and copy its value.\n\n" +
			"- **Using Cookie Editor Extension:**\n" +
			"  - Download the [Cookie Editor extension](https://cookie-editor.com/) for your browser.\n" +
			"  - Log in to Activision.\n" +
			"  - Use the extension to find and copy the \"ACT_SSO_COOKIE\" value.\n\n",

		"## Setting up your EZ-Captcha API Key:\n" +
			"To use your own EZ-Captcha API key and control your account check frequency:\n\n" +
			"1. Visit [EZ-Captcha's website](https://ez-captcha.com) to register and obtain an API key.\n" +
			"2. Use the `/setcaptchaservice` command to set your personal API key.\n" +
			"3. Use the `/setcheckinterval` command to set your preferred account check frequency (in minutes).\n\n" +
			"Note: Using your own API key allows you to control the frequency of checks and manage your usage.",
	}

	for partIndex, part := range helpApiGuide {
		var err error
		if partIndex == 0 {
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: part,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		} else {
			_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: part,
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}

		if err != nil {
			logger.Log.WithError(err).Error("Error responding to help api command")
			return
		}
	}
}
