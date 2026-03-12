// Comprehensive test suite for PromptMan Daemon API collection.
// Keys match folder/request-id paths from collection YAML.
// Requires a running daemon with dev environment (port + token).

module.exports = {

  // ── Root (no prefix) ─────────────────────────────────────

  "daemon-status": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Response is JSON", function () {
      var body = pm.response.json();
      pm.expect(body).to.be.an("object");
    });
  },

  // ── collections/ folder ──────────────────────────────────

  "collections/list-collections": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns array of collections", function () {
      var envelope = pm.response.json();
      pm.expect(envelope).to.have.property("data");
      pm.expect(envelope.data).to.be.an("array");
    });

    pm.test("Has at least one collection", function () {
      var data = pm.response.json().data;
      pm.expect(data.length).to.be.above(0);
    });
  },

  "collections/get-collection": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns collection data", function () {
      var envelope = pm.response.json();
      pm.expect(envelope).to.have.property("data");
    });
  },

  "collections/update-collection": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });
  },

  // ── environments/ folder ─────────────────────────────────

  "environments/list-environments": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns data array", function () {
      var envelope = pm.response.json();
      pm.expect(envelope).to.have.property("data");
    });
  },

  "environments/get-environment": function (pm) {
    pm.test("Status is 200 or 404", function () {
      var valid = pm.response.status === 200 || pm.response.status === 404;
      pm.expect(valid).to.equal(true);
    });
  },

  "environments/set-active-env": function (pm) {
    pm.test("Status is 200 or 400 or 404", function () {
      var valid = pm.response.status === 200 || pm.response.status === 400 || pm.response.status === 404;
      pm.expect(valid).to.equal(true);
    });
  },

  "environments/upsert-environment": function (pm) {
    pm.test("Status is 200 or 201", function () {
      var valid = pm.response.status === 200 || pm.response.status === 201;
      pm.expect(valid).to.equal(true);
    });

    pm.test("Response is JSON", function () {
      var body = pm.response.json();
      pm.expect(body).to.be.an("object");
    });
  },

  // ── history/ folder ──────────────────────────────────────

  "history/list-history": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns data", function () {
      var envelope = pm.response.json();
      pm.expect(envelope).to.have.property("data");
    });
  },

  "history/clear-history": function (pm) {
    pm.test("Status is 200 or 204", function () {
      var valid = pm.response.status === 200 || pm.response.status === 204;
      pm.expect(valid).to.equal(true);
    });
  },

  // ── requests/ folder ─────────────────────────────────────

  "requests/run-single-request": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns execution result", function () {
      var envelope = pm.response.json();
      pm.expect(envelope).to.have.property("data");
    });
  },

  "requests/run-collection": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns results", function () {
      var envelope = pm.response.json();
      pm.expect(envelope).to.have.property("data");
    });
  },

  // ── tests/ folder ────────────────────────────────────────

  "tests/run-tests": function (pm) {
    pm.test("Returns 200 or 422", function () {
      var valid = pm.response.status === 200 || pm.response.status === 422;
      pm.expect(valid).to.equal(true);
    });
  },

  "tests/list-test-results": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });
  },

  "tests/get-test-result": function (pm) {
    pm.test("Status is 200 or 404", function () {
      var valid = pm.response.status === 200 || pm.response.status === 404;
      pm.expect(valid).to.equal(true);
    });
  },

  // ── stress/ folder ───────────────────────────────────────

  "stress/run-stress": function (pm) {
    pm.test("Returns a result", function () {
      var valid = pm.response.status === 200 || pm.response.status === 400;
      pm.expect(valid).to.equal(true);
    });
  },

  "stress/list-stress-results": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });
  },

  "stress/get-stress-result": function (pm) {
    pm.test("Status is 200 or 404", function () {
      var valid = pm.response.status === 200 || pm.response.status === 404;
      pm.expect(valid).to.equal(true);
    });
  },
};
