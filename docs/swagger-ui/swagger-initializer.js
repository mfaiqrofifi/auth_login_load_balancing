window.onload = function () {
  const authStorageKey = "auth-login-load-balancing.access-token";

  function extractRawToken(token) {
    if (!token) {
      return "";
    }

    const trimmed = token.trim();
    if (trimmed.toLowerCase().startsWith("bearer ")) {
      return trimmed.slice(7).trim();
    }

    return trimmed;
  }

  function storeAccessToken(token) {
    const rawToken = extractRawToken(token);
    if (!rawToken) {
      return;
    }

    window.localStorage.setItem(authStorageKey, rawToken);
  }

  function clearAccessToken() {
    window.localStorage.removeItem(authStorageKey);
  }

  function readAccessToken() {
    return window.localStorage.getItem(authStorageKey) || "";
  }

  const ui = SwaggerUIBundle({
    url: "./openapi.yaml",
    dom_id: "#swagger-ui",
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout",
    persistAuthorization: true,
    requestInterceptor: function (request) {
      const token = extractRawToken(readAccessToken());
      const targetUrl = request.url || "";
      request.credentials = "include";

      if (token && !request.headers.Authorization && !targetUrl.endsWith("/auth/login") && !targetUrl.endsWith("/auth/register")) {
        request.headers.Authorization = "Bearer " + token;
      }

      return request;
    },
    responseInterceptor: function (response) {
      try {
        const targetUrl = response.url || "";
        const body = response.text ? JSON.parse(response.text) : null;

        if ((targetUrl.endsWith("/auth/login") || targetUrl.endsWith("/auth/refresh")) && body && body.access_token) {
          const rawToken = extractRawToken(body.access_token);
          storeAccessToken(rawToken);
          ui.preauthorizeApiKey("bearerAuth", rawToken);
        }

        if ((targetUrl.endsWith("/auth/logout") || targetUrl.endsWith("/auth/logout-all")) && response.status >= 200 && response.status < 300) {
          clearAccessToken();
          ui.authActions.logout(["bearerAuth"]);
        }
      } catch (error) {
        console.warn("Swagger auth helper could not parse response", error);
      }

      return response;
    }
  });

  window.ui = ui;

  const savedToken = extractRawToken(readAccessToken());
  if (savedToken) {
    ui.preauthorizeApiKey("bearerAuth", savedToken);
  }
};
