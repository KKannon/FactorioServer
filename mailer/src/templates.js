import path from "node:path";
import { readFile } from "node:fs/promises";

export const eventDefinitions = Object.freeze({
  MailIntegrationTest: { template: "system", action: "Teste de integração de e-mail", resource: "StupidMailCenter", status: "Concluído" },
  MonitoringAlert: { template: "system", action: "Alerta de monitoramento", resource: "Servidor Factorio", status: "Atenção necessária" },
  InstallPlayerBridge: { template: "system", action: "Bridge de inteligência de jogadores instalado", resource: "Servidor Factorio", status: "Reinicialização necessária" },
  RemoveSave: { template: "world", action: "Exclusão segura de mundo", resource: "Save Factorio", status: "Concluído com backup" },
  DeleteMod: { template: "system", action: "Mod removido", resource: "Mod Factorio", status: "Concluído" },
  DeleteAllMods: { template: "system", action: "Todos os mods removidos", resource: "Mods Factorio", status: "Concluído" },
  ModPackDelete: { template: "system", action: "Modpack removido", resource: "Modpack Factorio", status: "Concluído" },
  LoadModPack: { template: "system", action: "Modpack aplicado ao servidor", resource: "Modpack Factorio", status: "Concluído" },
  UserWelcome: { template: "welcome", action: "Bem-vindo ao Factorio Server Manager", resource: "Factorio Server Manager", status: "Conta conectada" },
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
  welcome: { name: "factorio-user-welcome-v1", subject: "Bem-vindo ao Factorio Server Manager", file: "welcome" },
  lifecycle: { name: "factorio-server-lifecycle-v1", subject: "[Factorio] {{{ACTION}}}", file: "lifecycle" },
  world: { name: "factorio-world-operation-v1", subject: "[Factorio] {{{ACTION}}}", file: "world" },
  access: { name: "factorio-access-change-v1", subject: "[Factorio] Alteração de acesso", file: "access" },
  system: { name: "factorio-system-change-v1", subject: "[Factorio] {{{ACTION}}}", file: "system" },
});

export async function loadTemplates(options = {}) {
  const templateDir = options.templateDir ?? process.env.MAILER_TEMPLATE_DIR ?? "/app/templates";
  const templates = {};
  for (const [key, definition] of Object.entries(templateDefinitions)) {
    const [html, text] = await Promise.all([
      readFile(path.join(templateDir, `${definition.file}.html`), "utf8"),
      readFile(path.join(templateDir, `${definition.file}.txt`), "utf8"),
    ]);
    templates[key] = Object.freeze({
      subject: definition.subject,
      html,
      text,
    });
  }
  return Object.freeze(templates);
}

export function templateDefinitionFor(event) {
  return eventDefinitions[event];
}

export function renderTemplate(template, variables) {
  const render = (source) => source.replace(/\{\{\{([A-Z][A-Z0-9_]*)\}\}\}/g, (_, name) => {
    if (!(name in variables)) throw new Error(`missing_template_variable_${name}`);
    return String(variables[name]);
  });
  const rendered = {
    subject: render(template.subject),
    html: render(template.html),
    text: render(template.text),
  };
  if (/\{\{\{[A-Z][A-Z0-9_]*\}\}\}/.test(Object.values(rendered).join("\n"))) {
    throw new Error("unresolved_template_variable");
  }
  return rendered;
}
