import path from "node:path";
import { readFile, rename, writeFile } from "node:fs/promises";
import { Template } from "lib-stupidmailjavascript";

export const eventDefinitions = Object.freeze({
  StartServer: { template: "lifecycle", action: "Inicialização do servidor solicitada", resource: "Servidor Factorio", status: "Em andamento" },
  StopServer: { template: "lifecycle", action: "Parada do servidor solicitada", resource: "Servidor Factorio", status: "Em andamento" },
  RestartServer: { template: "lifecycle", action: "Reinicialização do servidor solicitada", resource: "Servidor Factorio", status: "Em andamento" },
  KillServer: { template: "lifecycle", action: "Processo do servidor encerrado", resource: "Servidor Factorio", status: "Concluído" },
  CreateWorld: { template: "world", action: "Novo mundo criado", resource: "Mundo Factorio", status: "Concluído" },
  CreateSaveBackup: { template: "world", action: "Backup de mundo criado", resource: "Save Factorio", status: "Concluído" },
  RestoreSaveBackup: { template: "world", action: "Backup de mundo restaurado", resource: "Save Factorio", status: "Concluído" },
  RemoveSaveBackup: { template: "world", action: "Backup de mundo removido", resource: "Save Factorio", status: "Concluído" },
  RenameSave: { template: "world", action: "Save renomeado", resource: "Save Factorio", status: "Concluído" },
  AddWhitelistPlayer: { template: "access", action: "Jogador adicionado à whitelist", resource: "Controle de jogadores", status: "Concluído" },
  RemoveWhitelistPlayer: { template: "access", action: "Jogador removido da whitelist", resource: "Controle de jogadores", status: "Concluído" },
  AddAdmin: { template: "access", action: "Administrador do jogo adicionado", resource: "Controle de jogadores", status: "Concluído" },
  RemoveAdmin: { template: "access", action: "Administrador do jogo removido", resource: "Controle de jogadores", status: "Concluído" },
  AddBan: { template: "access", action: "Jogador banido", resource: "Controle de jogadores", status: "Concluído" },
  RemoveBan: { template: "access", action: "Banimento removido", resource: "Controle de jogadores", status: "Concluído" },
  UpdateWhitelistPolicy: { template: "access", action: "Política de whitelist alterada", resource: "Controle de jogadores", status: "Concluído" },
  InstallFactorioVersion: { template: "system", action: "Versão do Factorio instalada", resource: "Instalação do Factorio", status: "Concluído" },
  UpdateServerSettings: { template: "system", action: "Configurações do servidor alteradas", resource: "Configurações do servidor", status: "Concluído" },
});

const templateDefinitions = Object.freeze({
  lifecycle: { name: "factorio-server-lifecycle-v1", subject: "[Factorio] {{{ACTION}}}", file: "lifecycle" },
  world: { name: "factorio-world-operation-v1", subject: "[Factorio] {{{ACTION}}}", file: "world" },
  access: { name: "factorio-access-change-v1", subject: "[Factorio] Alteração de acesso", file: "access" },
  system: { name: "factorio-system-change-v1", subject: "[Factorio] {{{ACTION}}}", file: "system" },
});

export async function ensureTemplates(service, options = {}) {
  const dataDir = options.dataDir ?? process.env.MAILER_DATA_DIR ?? "/data";
  const templateDir = options.templateDir ?? process.env.MAILER_TEMPLATE_DIR ?? "/app/templates";
  const registryPath = path.join(dataDir, "template-registry-v1.json");
  let registry = {};
  try {
    registry = JSON.parse(await readFile(registryPath, "utf8"));
  } catch (error) {
    if (error?.code !== "ENOENT") throw error;
  }

  for (const [key, definition] of Object.entries(templateDefinitions)) {
    if (typeof registry[key] === "string" && registry[key]) continue;
    const template = await Template.fromFiles({
      name: definition.name,
      subject: definition.subject,
      htmlPath: path.join(templateDir, `${definition.file}.html`),
      textPath: path.join(templateDir, `${definition.file}.txt`),
    });
    const created = await service.createTemplate(template);
    registry[key] = created.id;
    const temporaryPath = `${registryPath}.tmp`;
    await writeFile(temporaryPath, `${JSON.stringify(registry, null, 2)}\n`, { mode: 0o600 });
    await rename(temporaryPath, registryPath);
  }
  return Object.freeze(registry);
}

export function templateDefinitionFor(event) {
  return eventDefinitions[event];
}
