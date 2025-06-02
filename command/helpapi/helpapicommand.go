package helpapi

import (
	"fmt"
	"strings"

	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bradselph/CODStatusBot/services"
	"github.com/bwmarrin/discordgo"
)

func CommandHelpApi(s *discordgo.Session, i *discordgo.InteractionCreate) {
	logger.Log.Info("Received help command")

	var enabledServices []string
	if services.IsServiceEnabled("capsolver") {
		enabledServices = append(enabledServices, "Capsolver")
	}
	if services.IsServiceEnabled("ezcaptcha") {
		enabledServices = append(enabledServices, "EZ-Captcha")
	}
	if services.IsServiceEnabled("2captcha") {
		enabledServices = append(enabledServices, "2captcha")
	}

	helpApiGuide := []string{
		"CODStatusBot Help Guide\n\n" +
			"To add your Call of Duty account to the bot, you'll need to obtain your SSO (Single Sign-On) cookie and set up a captcha service. Here's how:\n\n" +
			"1. **Getting Your SSO Cookie:**\n" +
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

		"## Setting up your Captcha Service:\n" +
			fmt.Sprintf("Currently available services: %s\n\n", strings.Join(enabledServices, ", ")),
	}

	if services.IsServiceEnabled("capsolver") {
		helpApiGuide = append(helpApiGuide,
			"## Setting up Capsolver (Recommended):\n"+
				"1. Visit [Capsolver's website](https://dashboard.capsolver.com/passport/register?inviteCode=6YjROhACQnvP) to register.\n"+
				"2. Purchase credits (as low as $0.001 per solve).\n"+
				"3. Use the `/setcaptchaservice` command with `capsolver` as the provider.\n\n")
	}

	if services.IsServiceEnabled("ezcaptcha") {
		helpApiGuide = append(helpApiGuide,
			"## Setting up EZ-Captcha:\n"+
				"1. Visit [EZ-Captcha's website](https://dashboard.ez-captcha.com/#/register?inviteCode=uyNrRgWlEKy) to register.\n"+
				"2. Request a free trial of 10,000 tokens.\n"+
				"3. Use the `/setcaptchaservice` command with `ezcaptcha` as the provider.\n\n")
	}

	if services.IsServiceEnabled("2captcha") {
		helpApiGuide = append(helpApiGuide,
			"## Setting up 2Captcha:\n"+
				"1. Visit [2Captcha's website](https://2captcha.com/) to register.\n"+
				"2. Purchase credits for your account.\n"+
				"3. Use the `/setcaptchaservice` command with `2captcha` as the provider.\n\n")
	}

	helpApiGuide = append(helpApiGuide,
		"## Additional Information:\n"+
			"• Use `/setcheckinterval` to customize your account check frequency.\n"+
			"• The bot will notify you when your captcha balance is running low.\n"+
			"• If you need to switch services, use `/setcaptchaservice` again with the new provider.\n\n")

	for partIndex, part := range helpApiGuide {
		var err error
		if partIndex == 0 {
			err = services.RespondWithPreference(s, i, part, nil, false)
		} else {
			_, err = services.FollowupWithPreference(s, i, part, nil, nil, false)
		}

		if err != nil {
			logger.Log.WithError(err).Error("Error responding to help api command")
			return
		}
	}
}
