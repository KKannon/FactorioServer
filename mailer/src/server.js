import { createServer as createHttpServer } from "node:http";
import { timingSafeEqual } from "node:crypto";
import { pathToFileURL } from "node:url";
import { StupidMailCenterService } from "lib-stupidmailjavascript";
import { ensureTemplates, templateDefinitionFor } from "./templates.js";

const maximumBodyBytes = 32 * 1024;

function secureEqual(left, right) {
  const first = Buffer.from(left ?? "");
  const second = Buffer.from(right ?? "");
  return first.length === second.length && first.length > 0 && timingSafeEqual(first, second);
}

function escapeHtml(value) {
  return String(value).replace(/[&<>"']/g, (character) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  })[character]);
}

function requiredString(value, field, maximum = 256) {
  if (typeof value !== "string" || !value.trim() || value.length > maximum) {
    throw new Error(`invalid_${field}`);
  }
  return value.trim();
}

async function readJson(request) {
  const chunks = [];
  let length = 0;
  for await (const chunk of request) {
    length += chunk.length;
    if (length > maximumBodyBytes) throw new Error("body_too_large");
    chunks.push(chunk);
  }
  return JSON.parse(Buffer.concat(chunks).toString("utf8"));
}

export function variablesFor(payload, definition, options = {}) {
  const actor = requiredString(payload.actorName, "actor_name");
  const role = requiredString(payload.actorRole, "actor_role", 100);
  const resource = requiredString(payload.resource || definition.resource, "resource");
  const occurredAt = new Date(requiredString(payload.occurredAt, "occurred_at", 100));
  if (Number.isNaN(occurredAt.getTime())) throw new Error("invalid_occurred_at");
  const time = new Intl.DateTimeFormat("pt-BR", {
    dateStyle: "long", timeStyle: "short", timeZone: options.timezone ?? "America/Sao_Paulo",
  }).format(occurredAt);
  const panelUrl = options.panelUrl ?? "https://factorio.stupidll.com";
  return {
    ACTION: definition.action,
    STATUS: definition.status,
    ACTOR_HTML: escapeHtml(actor),
    ACTOR_TEXT: actor,
    ROLE_HTML: escapeHtml(role),
    ROLE_TEXT: role,
    RESOURCE_HTML: escapeHtml(resource),
    RESOURCE_TEXT: resource,
    TIME_HTML: escapeHtml(time),
    TIME_TEXT: time,
    PANEL_URL: panelUrl,
  };
}

export function createMailerServer({ service, registry, internalToken, from, panelUrl, timezone }) {
  if (!internalToken) throw new Error("STUPID_MAIL_INTERNAL_TOKEN não está configurado");
  return createHttpServer(async (request, response) => {
    response.setHeader("Content-Type", "application/json; charset=utf-8");
    response.setHeader("Cache-Control", "no-store");
    if (request.method === "GET" && request.url === "/health") {
      response.writeHead(200).end('{"status":"ok"}');
      return;
    }
    if (request.method !== "POST" || request.url !== "/notify") {
      response.writeHead(404).end('{"error":"not_found"}');
      return;
    }
    if (!secureEqual(request.headers.authorization, `Bearer ${internalToken}`)) {
      response.writeHead(401).end('{"error":"unauthorized"}');
      return;
    }
    try {
      const payload = await readJson(request);
      const event = requiredString(payload.event, "event", 100);
      const definition = templateDefinitionFor(event);
      if (!definition) throw new Error("unsupported_event");
      const templateId = registry[definition.template];
      if (!templateId) throw new Error("missing_template");
      const result = await service.sendTemplate({
        from,
        to: requiredString(payload.recipient, "recipient", 320),
        templateId,
        templateVariables: variablesFor(payload, definition, { panelUrl, timezone }),
        idempotencyKey: requiredString(payload.idempotencyKey, "idempotency_key"),
      });
      response.writeHead(202).end(JSON.stringify({ id: result.id, status: result.status }));
    } catch (error) {
      const safe = { name: error?.name ?? "Error", code: error?.code, statusCode: error?.statusCode, requestId: error?.requestId };
      console.error("mail notification failed", safe);
      response.writeHead(502).end('{"error":"delivery_failed"}');
    }
  });
}

export async function start() {
  const service = new StupidMailCenterService({ timeoutMs: 10_000, maxRetries: 3 });
  const registry = await ensureTemplates(service);
  const server = createMailerServer({
    service,
    registry,
    internalToken: process.env.STUPID_MAIL_INTERNAL_TOKEN,
    from: process.env.STUPID_MAIL_FROM ?? "noreply@stupidll.com",
    panelUrl: process.env.STUPID_MAIL_PANEL_URL ?? "https://factorio.stupidll.com",
    timezone: process.env.TZ ?? "America/Sao_Paulo",
  });
  const port = Number(process.env.MAILER_PORT ?? 8080);
  server.listen(port, "0.0.0.0", () => console.log(`Factorio mailer listening on port ${port}`));
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  start().catch((error) => {
    console.error("mailer startup failed", { name: error?.name, code: error?.code, statusCode: error?.statusCode, requestId: error?.requestId });
    process.exitCode = 1;
  });
}
