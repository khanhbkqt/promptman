// Comprehensive test suite for JSONPlaceholder API collection.
// Keys match folder/request-id paths from collectionYAML.

module.exports = {

  // ── Root requests (no prefix) ────────────────────────────

  "get-all-posts": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns an array of posts", function () {
      var posts = pm.response.json();
      pm.expect(posts).to.be.an("array");
    });

    pm.test("Has 100 posts", function () {
      var posts = pm.response.json();
      pm.expect(posts.length).to.equal(100);
    });

    pm.test("First post has required fields", function () {
      var post = pm.response.json()[0];
      pm.expect(post).to.have.property("id");
      pm.expect(post).to.have.property("title");
      pm.expect(post).to.have.property("body");
      pm.expect(post).to.have.property("userId");
    });

    pm.test("Response time is under 5 seconds", function () {
      pm.expect(pm.response.time).to.be.below(5000);
    });
  },

  "get-post-by-id": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns correct post", function () {
      var post = pm.response.json();
      pm.expect(post.id).to.equal(1);
    });

    pm.test("Post has all fields", function () {
      var post = pm.response.json();
      pm.expect(post).to.have.property("userId");
      pm.expect(post).to.have.property("title");
      pm.expect(post).to.have.property("body");
    });

    pm.test("Title is a string", function () {
      var post = pm.response.json();
      pm.expect(post.title).to.be.a("string");
    });
  },

  "create-post": function (pm) {
    pm.test("Status is 201 Created", function () {
      pm.expect(pm.response.status).to.equal(201);
    });

    pm.test("Response contains the sent title", function () {
      var body = pm.response.json();
      pm.expect(body.title).to.equal("Hello PromptMan");
    });

    pm.test("Response contains userId", function () {
      var body = pm.response.json();
      pm.expect(body.userId).to.equal(1);
    });

    pm.test("Response has an id field", function () {
      var body = pm.response.json();
      pm.expect(body).to.have.property("id");
    });
  },

  "update-post": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Title was updated", function () {
      var body = pm.response.json();
      pm.expect(body.title).to.equal("Updated Title");
    });

    pm.test("Body was updated", function () {
      var body = pm.response.json();
      pm.expect(body.body).to.equal("Updated body content");
    });

    pm.test("ID is preserved", function () {
      var body = pm.response.json();
      pm.expect(body.id).to.equal(1);
    });
  },

  "delete-post": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Response body is empty object", function () {
      var body = pm.response.text();
      pm.expect(body).to.be.a("string");
    });
  },

  // ── comments/ folder ─────────────────────────────────────

  "comments/get-comments": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns an array of comments", function () {
      var comments = pm.response.json();
      pm.expect(comments).to.be.an("array");
    });

    pm.test("Has 5 comments for post 1", function () {
      var comments = pm.response.json();
      pm.expect(comments.length).to.equal(5);
    });

    pm.test("Comment has required fields", function () {
      var comment = pm.response.json()[0];
      pm.expect(comment).to.have.property("email");
      pm.expect(comment).to.have.property("body");
      pm.expect(comment).to.have.property("name");
    });
  },

  "comments/get-comment-by-id": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns correct comment", function () {
      var comment = pm.response.json();
      pm.expect(comment.id).to.equal(1);
      pm.expect(comment.postId).to.equal(1);
    });
  },

  // ── users/ folder ────────────────────────────────────────

  "users/get-all-users": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns 10 users", function () {
      var users = pm.response.json();
      pm.expect(users).to.be.an("array");
      pm.expect(users.length).to.equal(10);
    });

    pm.test("User has contact info", function () {
      var user = pm.response.json()[0];
      pm.expect(user).to.have.property("name");
      pm.expect(user).to.have.property("email");
      pm.expect(user).to.have.property("phone");
    });
  },

  "users/get-user-by-id": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns correct user", function () {
      var user = pm.response.json();
      pm.expect(user.id).to.equal(1);
      pm.expect(user.name).to.equal("Leanne Graham");
    });

    pm.test("User has address object", function () {
      var user = pm.response.json();
      pm.expect(user).to.have.property("address");
      pm.expect(user.address).to.have.property("city");
    });

    pm.test("User has company object", function () {
      var user = pm.response.json();
      pm.expect(user).to.have.property("company");
      pm.expect(user.company).to.have.property("name");
    });
  },

  "users/get-user-posts": function (pm) {
    pm.test("Status is 200", function () {
      pm.expect(pm.response.status).to.equal(200);
    });

    pm.test("Returns array of posts by user 1", function () {
      var posts = pm.response.json();
      pm.expect(posts).to.be.an("array");
      pm.expect(posts.length).to.be.above(0);
    });

    pm.test("All posts belong to user 1", function () {
      var posts = pm.response.json();
      for (var i = 0; i < posts.length; i++) {
        pm.expect(posts[i].userId).to.equal(1);
      }
    });
  },
};
