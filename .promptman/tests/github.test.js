// Comprehensive test suite for GitHub Public API collection.
// Keys match folder/request-id paths from collection YAML.
// Note: folder is "users-api" not "users" per the YAML.

module.exports = {

  // ── Root requests (no prefix) ────────────────────────────

  "get-rate-limit": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Has rate limit info", function () {
      var body = pm.response.json();
      pm.expect(body).to.have.property("rate");
      pm.expect(body).to.have.property("resources");
    });

    pm.test("Rate has limit and remaining", function () {
      var rate = pm.response.json().rate;
      pm.expect(rate).to.have.property("limit");
      pm.expect(rate).to.have.property("remaining");
    });
  },

  // ── repos/ folder ────────────────────────────────────────

  "repos/search-repos": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns search results", function () {
      var body = pm.response.json();
      pm.expect(body).to.have.property("total_count");
      pm.expect(body).to.have.property("items");
      pm.expect(body.items).to.be.an("array");
    });
  },

  "repos/get-repo": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns the Go repo", function () {
      var repo = pm.response.json();
      pm.expect(repo.full_name).to.equal("golang/go");
    });

    pm.test("Repo has key metadata", function () {
      var repo = pm.response.json();
      pm.expect(repo).to.have.property("description");
      pm.expect(repo).to.have.property("stargazers_count");
      pm.expect(repo).to.have.property("language");
    });

    pm.test("Language is Go", function () {
      var repo = pm.response.json();
      pm.expect(repo.language).to.equal("Go");
    });

    pm.test("Has positive star count", function () {
      var repo = pm.response.json();
      pm.expect(repo.stargazers_count).to.be.above(0);
    });
  },

  "repos/list-repo-issues": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns array of issues", function () {
      var issues = pm.response.json();
      pm.expect(issues).to.be.an("array");
    });

    pm.test("Issues have required fields", function () {
      var issues = pm.response.json();
      if (issues.length > 0) {
        pm.expect(issues[0]).to.have.property("title");
        pm.expect(issues[0]).to.have.property("state");
        pm.expect(issues[0]).to.have.property("number");
      }
    });

    pm.test("Returns at most 5 issues", function () {
      var issues = pm.response.json();
      pm.expect(issues.length).to.be.below(6);
    });
  },

  "repos/list-repo-commits": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns array of commits", function () {
      var commits = pm.response.json();
      pm.expect(commits).to.be.an("array");
    });

    pm.test("Commit has sha and message", function () {
      var commits = pm.response.json();
      if (commits.length > 0) {
        pm.expect(commits[0]).to.have.property("sha");
        pm.expect(commits[0]).to.have.property("commit");
        pm.expect(commits[0].commit).to.have.property("message");
      }
    });

    pm.test("Returns at most 5 commits", function () {
      var commits = pm.response.json();
      pm.expect(commits.length).to.be.below(6);
    });
  },

  // ── users-api/ folder ────────────────────────────────────

  "users-api/get-user": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns octocat profile", function () {
      var user = pm.response.json();
      pm.expect(user.login).to.equal("octocat");
    });

    pm.test("User has profile fields", function () {
      var user = pm.response.json();
      pm.expect(user).to.have.property("avatar_url");
      pm.expect(user).to.have.property("public_repos");
      pm.expect(user).to.have.property("followers");
    });

    pm.test("User type is User", function () {
      var user = pm.response.json();
      pm.expect(user.type).to.equal("User");
    });
  },

  "users-api/list-user-repos": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns array of repos", function () {
      var repos = pm.response.json();
      pm.expect(repos).to.be.an("array");
    });

    pm.test("Returns at most 5 repos", function () {
      var repos = pm.response.json();
      pm.expect(repos.length).to.be.below(6);
    });

    pm.test("Repos have standard fields", function () {
      var repos = pm.response.json();
      if (repos.length > 0) {
        pm.expect(repos[0]).to.have.property("name");
        pm.expect(repos[0]).to.have.property("full_name");
        pm.expect(repos[0]).to.have.property("html_url");
      }
    });
  },
};
