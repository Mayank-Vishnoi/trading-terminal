## Current flow:

1. Get the auth code from [here](https://api.upstox.com/v2/login/authorization/dialog?client_id={}&redirect_uri={}&response_type=code).
2. Hit the `getToken` API.
3. Save the `ACCESS_TOKEN` manually in the `.env` file.
4. Restart the server. It should be good until the next opening of the market!