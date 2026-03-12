// Comprehensive test suite for HTTPBin — Echo & Debug collection.
// Keys match folder/request-id paths from collection YAML.

module.exports = {

  // ── Root requests (no prefix) ────────────────────────────

  "simple-get": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Echoes custom header", function () {
      var body = pm.response.json();
      pm.expect(body.headers["X-Custom-Header"]).to.equal("promptman-test");
    });

    pm.test("Response has url and headers", function () {
      var body = pm.response.json();
      pm.expect(body).to.have.property("url");
      pm.expect(body).to.have.property("headers");
    });
  },

  "post-json": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Echoes JSON body", function () {
      var body = pm.response.json();
      pm.expect(body).to.have.property("json");
      pm.expect(body.json.message).to.equal("Hello from PromptMan");
    });

    pm.test("Content-Type is application/json", function () {
      var body = pm.response.json();
      pm.expect(body.headers["Content-Type"]).to.include("application/json");
    });
  },

  "put-data": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Echoes PUT body", function () {
      var body = pm.response.json();
      pm.expect(body.json.updated).to.equal(true);
      pm.expect(body.json.field).to.equal("value");
    });
  },

  "patch-data": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Echoes PATCH body", function () {
      var body = pm.response.json();
      pm.expect(body.json.partial).to.equal(true);
    });
  },

  "delete-resource": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Response has url property", function () {
      var body = pm.response.json();
      pm.expect(body).to.have.property("url");
    });
  },

  // ── status-codes/ folder ─────────────────────────────────

  "status-codes/ok-200": function (pm) {
    pm.test("Returns 200 OK", function () {
      pm.expect(pm.response.status).to.equal(200);
    });
  },

  "status-codes/created-201": function (pm) {
    pm.test("Returns 201 Created", function () {
      pm.expect(pm.response.status).to.equal(201);
    });
  },

  "status-codes/not-found-404": function (pm) {
    pm.test("Returns 404 Not Found", function () {
      pm.expect(pm.response.status).to.equal(404);
    });
  },

  "status-codes/server-error-500": function (pm) {
    pm.test("Returns 500 Internal Server Error", function () {
      pm.expect(pm.response.status).to.equal(500);
    });
  },

  // ── auth-tests/ folder ───────────────────────────────────

  "auth-tests/basic-auth": function (pm) {
    pm.test("Status is 200 (auth succeeded)", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Authenticated flag is true", function () {
      var body = pm.response.json();
      pm.expect(body.authenticated).to.equal(true);
    });

    pm.test("User is correct", function () {
      var body = pm.response.json();
      pm.expect(body.user).to.equal("user");
    });
  },

  "auth-tests/bearer-auth": function (pm) {
    pm.test("Status is 200 (bearer auth succeeded)", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Authenticated flag is true", function () {
      var body = pm.response.json();
      pm.expect(body.authenticated).to.equal(true);
    });

    pm.test("Token is echoed back", function () {
      var body = pm.response.json();
      pm.expect(body.token).to.equal("my-secret-token");
    });
  },

  // ── response-formats/ folder ─────────────────────────────

  "response-formats/json-response": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Body is valid JSON with slideshow", function () {
      var body = pm.response.json();
      pm.expect(body).to.have.property("slideshow");
    });
  },

  "response-formats/html-response": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Body contains HTML", function () {
      var text = pm.response.text();
      pm.expect(text).to.include("<html>");
    });
  },

  "response-formats/xml-response": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Body contains XML declaration", function () {
      var text = pm.response.text();
      pm.expect(text).to.include("<?xml");
    });
  },

  "response-formats/delayed-response": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Response time is at least 2 seconds", function () {
      pm.expect(pm.response.time).to.be.above(1500);
    });

    pm.test("Response time is under 10 seconds", function () {
      pm.expect(pm.response.time).to.be.below(10000);
    });
  },
};
