### Error Code

When requesting the interface, if the `errorId` is returned with a value of `1`, you can see the specific error information in `errorCode` and `errorDescription`.

```json
{
    "errorId": 1,
    "errorCode": "the error code",
    "errorDescription": "error description"
}
```

| **ErrorCode**                    | **ErrorDescription**           | **Description**                                                                                                                                                                        |
|---------------------------------|--------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ERROR_SERVICE_UNAVALIABLE        | service temporarily unavailable | It may be that the server is under high pressure. Please try again later. If it persists, please contact customer service.                                                         |
| ERROR_REQUIRED_FIELDS            | Please check the field: XXX    | Missing or incorrect fields. Please check carefully.                                                                                                                                 |
| ERROR_INTERNAL_SERVER_ERROR      | internal server error          | If it persists, please contact customer service.                                                                                                                                        |
| ERROR_IP_BANNED                  | Your IP has been blocked for 10 minutes | **A large number of errors occur in a short period of time** (including sending lots of requests when your balance is empty), and it will be automatically blocked for a few minutes. **Please stop your requests and check your error message, and wait a few minutes**. |
| ERROR_KEY_BANNED                 | Your client key has been banned | Your KEY has been banned. Please contact customer service.                                                                                                                             |
| ERROR_TASKID_INVALID             | Task ID does not exist or has expired | The wrong ID was requested, or the ID no longer exists.                                                                                                                               |
| ERROR_KEY_DOES_NOT_EXIST         | client key error               | Please check whether your clientKey key is correct. Get it in the personal center.                                                                                                    |
| ERROR_ZERO_BALANCE               | Insufficient account balance   | The account balance is not enough for consumption. Please recharge.                                                                                                                     |
| ERROR_TASK_NOT_SUPPORTED         | Task type not supported        | The verification code type is incorrect or not yet supported.                                                                                                                           |
| ERROR_RECAPTCHA_INVALID_SITEKEY  | Invalid target site SITEKEY    | Your points will not be deducted for this error. Please try again. Please check your sitekey.                                                                                           |
| INVALID_RECAPTCHA_SITEURL        | Invalid target site url        | Your points will not be deducted for this error. Please try again. Please check your website url.                                                                                     |
| ERROR_NO_SLOT_AVAILABLE          | Insufficient server resources, please try again later | Insufficient server resources. Please try again later.                                                                                                                                |