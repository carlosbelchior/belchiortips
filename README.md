# Belchior Tips - Telegram Bot 🚀

Este é um bot automatizado desenvolvido em Go (Golang) para o Telegram. Sua função principal é conectar-se à API da **Green365**, buscar oportunidades de apostas (tips) de um ou múltiplos robôs configurados, validar as oportunidades e enviá-las para um canal do Telegram. Ele também faz o acompanhamento dos resultados ao vivo e um fechamento diário do lucro/prejuízo.

## 🎯 Principais Funcionalidades

- **Múltiplos Bots:** Suporta a busca de oportunidades em vários bots da Green365 simultaneamente (`BOT_ID=123,456`).
- **Acompanhamento ao Vivo:** Quando um jogo finaliza (Green, Red, Meio Green ou Meio Red), o sistema automaticamente edita a mensagem no canal do Telegram para exibir o resultado.
- **Validador Opcional:** Integra-se à Bolsa de Aposta. Se a `BOLSADEAPOSTA_TOKEN` for configurada, o bot não envia a dica até validar que a partida realmente foi aberta na corretora.
- **Fechamento Diário Automático:** Todos os dias às **23:00**, o bot faz um levantamento completo:
  - Valida silenciosamente (sem gerar notificações) se as mensagens ainda existem no canal ou se foram excluídas pela administração. Entradas apagadas são **ignoradas** na contabilidade final.
  - Soma as Tips, Greens, Reds e o Saldo (Lucro/Prejuízo).
  - Separa as estatísticas pelos mercados, como **Ambos Marcam** e **Over/Under**.
  - Envia o relatório detalhado no próprio Telegram.
- **Tratamento Inteligente de Mercado:** Consegue ler parâmetros técnicos ("Yes"/"No") da API e traduzi-los para o público final (ex: "Ambos Marcam SIM", "Ambos Marcam NÃO").

## ⚙️ Configuração (.env)

O sistema depende de variáveis de ambiente. Você deve criar um arquivo `.env` na raiz do projeto com base no `.env.example`:

```env
# Credenciais da Green365
GREEN365_EMAIL=seu_email@gmail.com
GREEN365_PASSWORD=sua_senha
BOT_ID=123,456 # Pode ser um ID único ou vários separados por vírgula

# Credenciais do Telegram
TELEGRAM_BOT_TOKEN=seu_bot_token_aqui
TELEGRAM_CHAT_ID=-100XXXXXXXXXX

# Bolsa de Aposta (Opcional - para validação de jogos ao vivo)
BOLSADEAPOSTA_TOKEN=seu_token_da_bolsa
BOLSADEAPOSTA_COOKIE=seus_cookies_aqui

# Regras de Negócio
MIN_ODD=1.70
```

## 🛠 Como executar

Certifique-se de ter o [Go](https://go.dev/) instalado.

1. Baixe as dependências:
```bash
go mod tidy
```

2. Execute o bot:
```bash
go run main.go
```

Ou, para compilar a versão de produção:
```bash
go build -o belchiortips main.go
./belchiortips
```

## 📂 Estrutura do Projeto

- `config/`: Gerenciamento das variáveis de ambiente (`.env`).
- `green365/`: Cliente HTTP e structs para conectar-se e autenticar-se na API da Green365.
- `report/`: Lógica do fechamento diário, armazenamento de estatísticas (`reports/*.json`) e formatação do sumário diário.
- `telegram/`: Wrapper da API do Telegram (envio, edição de mensagens, botões inline e tradução de flags/mercados).
- `tracker/`: Memória de curto prazo (`tracker_data.json`) usada para relacionar as `OpportunityID`s com os `MessageID`s do Telegram, possibilitando a edição futura da mensagem.
- `validator/`: Subsistema opcional que consulta a corretora Bolsa de Aposta para validar se a odd e a partida realmente abriram no mercado antes de notificar os clientes.

## ⚠️ Sobre o Validador de Mensagens
Para o envio correto dos relatórios diários, o sistema confia no fato de que bots no Telegram não possuem função nativa de leitura de histórico. Por isso, a rotina de validação simula uma edição do botão da mensagem para descobrir se ela ainda existe no chat (caso contrário, a API do Telegram retorna um erro específico que o sistema intercepta). Isso evita que os usuários recebam notificações falsas durante o fechamento e garante dados estritos.