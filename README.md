This project implements a oauth server using ldap as backend by using two projects

- oauth-authenticator
- go-ldap-authenticator

Endpoints for OAuth Authorization

    access_token_url='BASE/oauth/token',
    access_token_params=None,
    authorize_url='BASE/oauth/authorize',
    authorize_params=None,
    api_base_url='BASE/api/v4/',
    
Der Scope ist profile

Unter der API Base URL ist der Endpoint ``user`` analog zur GitLab v4 API implementiert. Man erhält Daten zum soeben angemeldeten Nutzer.
