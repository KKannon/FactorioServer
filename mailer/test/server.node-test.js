import assert from "node:assert/strict";
import { once } from "node:events";
import { fileURLToPath } from "node:url";
import test from "node:test";
import { createMailerServer, variablesFor } from "../src/server.js";
import { eventDefinitions, loadTemplates, renderTemplate } from "../src/templates.js";

test("HTML variables are escaped while text remains readable", () => {
  const values = variablesFor({ actorName: "<Admin>", actorRole: "owner", resource: "save & world", occurredAt: "2026-09-25T12:00:00Z" }, eventDefinitions.CreateSaveBackup);
  assert.equal(values.ACTOR_HTML, "&lt;Admin&gt;");
  assert.equal(values.ACTOR_TEXT, "<Admin>");
  assert.equal(values.RESOURCE_HTML, "save &amp; world");
});

test("welcome template renders the first-login message without placeholders", async () => {
  const templateDir = fileURLToPath(new URL("../templates", import.meta.url));
  const templates = await loadTemplates({ templateDir });
  const definition = eventDefinitions.UserWelcome;
  const values = variablesFor({ actorName: "Player", actorRole: "member", resource: "Factorio Server Manager", occurredAt: "2026-09-25T12:00:00Z" }, definition);
  const rendered = renderTemplate(templates[definition.template], values);
  assert.match(rendered.subject, /Bem-vindo/);
  assert.match(rendered.html, /Bem-vindo, Player/);
  assert.match(rendered.text, /enviado uma única vez/);
  assert.doesNotMatch(`${rendered.subject}${rendered.html}${rendered.text}`, /\{\{\{/);
});

test("monitoring alerts use the backend-owned system template", () => {
  assert.equal(eventDefinitions.MonitoringAlert.template, "system");
  assert.match(eventDefinitions.MonitoringAlert.action, /monitoramento/i);
});

test("player bridge installation has a restart notification", () => {
  assert.equal(eventDefinitions.InstallPlayerBridge.template, "system");
  assert.match(eventDefinitions.InstallPlayerBridge.status, /reinicialização/i);
});

test("destructive administration events use server-owned templates", () => {
  for (const event of ["RemoveSave", "DeleteMod", "DeleteAllMods", "ModPackDelete", "LoadModPack"]) {
    assert.ok(eventDefinitions[event]);
    assert.match(eventDefinitions[event].resource, /Factorio/i);
  }
});

test("notification endpoint authenticates and delegates to the library service", async (t) => {
  const sent = [];
  const server = createMailerServer({
    service: { async sendEmail(input) { sent.push(input); return { id: "mail-1", status: "queued" }; } },
    templates: { lifecycle: { subject: "{{{ACTION}}}", html: "<p>{{{ACTOR_HTML}}}</p>", text: "{{{ACTOR_TEXT}}}" } },
    internalToken: "internal-test-token", from: "noreply@stupidll.com",
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
  assert.equal(sent[0].to, "admin@example.com");
  assert.match(sent[0].subject, /Inicialização/);
  assert.equal(sent[0].html, "<p>Admin</p>");
});
