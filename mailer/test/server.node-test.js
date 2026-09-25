import assert from "node:assert/strict";
import { once } from "node:events";
import test from "node:test";
import { createMailerServer, variablesFor } from "../src/server.js";
import { eventDefinitions } from "../src/templates.js";

test("HTML variables are escaped while text remains readable", () => {
  const values = variablesFor({ actorName: "<Admin>", actorRole: "owner", resource: "save & world", occurredAt: "2026-09-25T12:00:00Z" }, eventDefinitions.CreateSaveBackup);
  assert.equal(values.ACTOR_HTML, "&lt;Admin&gt;");
  assert.equal(values.ACTOR_TEXT, "<Admin>");
  assert.equal(values.RESOURCE_HTML, "save &amp; world");
});

test("notification endpoint authenticates and delegates to the library service", async (t) => {
  const sent = [];
  const server = createMailerServer({
    service: { async sendTemplate(input) { sent.push(input); return { id: "mail-1", status: "queued" }; } },
    registry: { lifecycle: "template-1" }, internalToken: "internal-test-token", from: "noreply@stupidll.com",
  });
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  t.after(() => server.close());
  const address = server.address();
  const payload = { event: "StartServer", recipient: "admin@example.com", actorName: "Admin", actorRole: "owner", resource: "Servidor Factorio", occurredAt: new Date().toISOString(), idempotencyKey: "factorio/test/1" };

  const unauthorized = await fetch(`http://127.0.0.1:${address.port}/notify`, { method: "POST", body: JSON.stringify(payload) });
  assert.equal(unauthorized.status, 401);
  const accepted = await fetch(`http://127.0.0.1:${address.port}/notify`, { method: "POST", headers: { Authorization: "Bearer internal-test-token" }, body: JSON.stringify(payload) });
  assert.equal(accepted.status, 202);
  assert.equal(sent.length, 1);
  assert.equal(sent[0].templateId, "template-1");
  assert.equal(sent[0].to, "admin@example.com");
});
