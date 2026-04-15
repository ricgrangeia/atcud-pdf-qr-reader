# Leitor de QR Code Fiscal ATCUD

**Versão:** 1.2.0 · **Licença:** MIT · **Autor:** [Ricardo Grangeia](https://ricardo.grangeia.pt)

Serviço HTTP escrito em Go para leitura e descodificação de **QR codes ATCUD** em documentos fiscais portugueses. Suporta **PDF e imagens**. Extrai o NIF do emitente, NIF do adquirente, tipo de documento, linhas de IVA por taxa e região fiscal, totais e muito mais — tudo em JSON estruturado. Opcionalmente identifica os nomes das entidades via serviço de IA local.

---

## Funcionalidades

- Recebe um ficheiro **PDF ou imagem** (JPEG, PNG, GIF, WEBP, TIFF) por HTTP
- Detecta e descodifica **todos os QR codes** em todas as páginas
- Filtra os que contêm um código **ATCUD** válido (especificação AT)
- Devolve JSON com os dados em bruto (`/scan`) ou totalmente estruturados (`/parse`)
- Modo **enriquecido** (`/parse/enriched`) — resolve automaticamente o nome do emitente e adquirente via serviço de NIF por IA
- Lookup de NIF avulso ou em bulk via `/api/v1/nif/lookup/bulk`
- Interface web integrada em português de Portugal com 3 modos de operação
- Documentação interactiva **OpenAPI 3.1** via Swagger UI
- Ficheiros apagados automaticamente após processamento — nenhum documento fica no servidor
- Pronto para Docker, Portainer e Traefik

---

## Privacidade e segurança dos ficheiros

**Nenhum ficheiro fica guardado no servidor.** O ciclo de vida de um documento enviado é o seguinte:

1. O PDF é escrito num ficheiro temporário (`os.CreateTemp`) — [`internal/infrastructure/pdf/scanner.go`](internal/infrastructure/pdf/scanner.go)
2. As páginas são renderizadas para PNG numa pasta temporária — [`internal/infrastructure/pdf/renderer.go`](internal/infrastructure/pdf/renderer.go)
3. Após extracção dos QR codes, ambos são apagados via `defer os.Remove` e `defer os.RemoveAll`
4. Imagens são processadas directamente em memória — sem escrita em disco
5. Os dados transitam apenas em memória — o servidor nunca persiste nem regista o conteúdo dos documentos

---

## Infográfico

![Fluxo de processamento](infographic.svg)

---

## Arquitectura

O projecto segue uma arquitectura **DDD simplificada**:

```
cmd/
  go_api/
    main.go                      ← ponto de entrada

internal/
  config/                        ← variáveis de ambiente
  domain/document/               ← entidades e regras de negócio
    qrcode.go                    ← entidade QRCode
    atcud.go                     ← detecção de ATCUD (regex)
    parsed_qrcode.go             ← documento fiscal estruturado (+ descricao)
    qrcode_parser.go             ← parser dos campos do QR (spec AT)
  application/document/          ← casos de uso
    service.go                   ← ScanPDF, ParsePDF, ScanImage, ParseImage
  infrastructure/pdf/            ← adaptadores externos
    renderer.go                  ← renderização de páginas (pdftoppm)
    scanner.go                   ← detecção de QR codes (gozxing)
  interfaces/http/               ← camada HTTP
    handler.go                   ← handlers PDF e imagem
    enriched_handler.go          ← handlers com enriquecimento NIF
    nif_handler.go               ← lookup de NIF (bulk, proxy para IA)
    router.go                    ← rotas e configuração
  ui/                            ← interface web embutida
    embed.go
    index.html
```

---

## Endpoints da API

| Método | Caminho | Descrição |
|--------|---------|-----------|
| `POST` | `/api/v1/document/scan` | PDF — conteúdo bruto dos QR codes com ATCUD |
| `POST` | `/api/v1/document/parse` | PDF — dados fiscais estruturados |
| `POST` | `/api/v1/document/parse/enriched` | PDF — dados fiscais + nome das entidades (IA) |
| `POST` | `/api/v1/image/scan` | Imagem — conteúdo bruto dos QR codes com ATCUD |
| `POST` | `/api/v1/image/parse` | Imagem — dados fiscais estruturados |
| `POST` | `/api/v1/image/parse/enriched` | Imagem — dados fiscais + nome das entidades (IA) |
| `POST` | `/api/v1/nif/lookup/bulk` | Resolve lista de NIFs para nomes (bulk, máx. 20) |
| `GET`  | `/api/v1/version` | Versão e autor |
| `GET`  | `/health` | Estado do serviço |
| `GET`  | `/docs` | Swagger UI (OpenAPI 3.1) |
| `GET`  | `/` | Interface web |

### Exemplo de resposta — `/api/v1/document/parse`

```json
{
  "total_qr_codes": 1,
  "parsed_count": 1,
  "documents": [
    {
      "numero_pagina": 1,
      "conteudo_bruto": "A:508136695*B:999999990*C:PT*D:FT*E:N*F:20250917*G:FT 2025A/341*H:KXTP8ZQ2-341*I1:PT*I7:142.68*I8:32.82*N:32.82*O:175.50*Q:pNaK*R:1287",
      "emitente": { "nif": "508136695" },
      "adquirente": { "nif": "999999990", "pais": "PT" },
      "documento": {
        "tipo_codigo": "FT", "tipo": "Fatura",
        "estado_codigo": "N", "estado": "Normal",
        "data": "2025-09-17", "identificador": "FT 2025A/341", "atcud": "KXTP8ZQ2-341"
      },
      "impostos": {
        "linhas": [{ "regiao": "Portugal Continental", "taxa": "Taxa Normal", "base_tributavel": 142.68, "valor_iva": 32.82 }],
        "total_imposto": 32.82, "retencao_fonte": 0
      },
      "totais": { "total_documento": 175.50 },
      "caracteres_assinatura": "pNaK",
      "numero_certificado": "1287",
      "informacoes_adicionais": ""
    }
  ]
}
```

### Exemplo de resposta — `/api/v1/document/parse/enriched`

```json
{
  "emitente": {
    "descricao": "EDP COMERCIAL - COMERCIALIZAÇÃO DE ENERGIA, S.A.",
    "nif": "503504564"
  },
  "adquirente": {
    "descricao": "ESCRITA GULOSA",
    "nif": "513830146",
    "pais": "PT"
  }
}
```

> O campo `descricao` só aparece nos endpoints `/enriched`. Nos endpoints normais é omitido.

---

## Como executar localmente

### Pré-requisitos

- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- Git

### 1. Clonar o repositório

```bash
git clone <url-do-repositório>
cd atcud-pdf-qr-code-reader
```

### 2. Configurar variáveis de ambiente

| Variável | Descrição | Obrigatória |
|----------|-----------|-------------|
| `PORT` | Porta HTTP do servidor (omissão: `8080`) | Não |
| `URL_HOST_DOMAIN` | Domínio público do serviço | Não |
| `GIN_MODE` | Modo do Gin (`debug` / `release`) | Não |
| `TOOL_SERVER_URL` | URL base do servidor de ferramentas de IA (lookup NIF) | Não |
| `TOOL_SERVER_API_KEY` | Chave `x-api-key` do servidor de ferramentas | Não |
| `PROXY_NETWORK_NAME` | Nome da rede Docker do Traefik | Sim (produção) |
| `AI_NETWORK_NAME` | Nome da rede Docker do servidor de IA | Não |

> `TOOL_SERVER_URL` e `TOOL_SERVER_API_KEY` são opcionais. Se não configurados, os endpoints `/enriched` e `/nif/lookup/bulk` devolvem `found: false` para NIFs não especiais.

### 3. Construir e executar

```bash
docker build -t go-api-app .

docker run --rm -p 8080:8080 go-api-app
```

### 4. Abrir no browser

| URL | O que abre |
|-----|-----------|
| http://localhost:8080/ | Interface web |
| http://localhost:8080/docs | Swagger UI |
| http://localhost:8080/openapi.json | Especificação OAS 3.1 |

---

## Implementação no Portainer

O `docker-compose.yml` usa variáveis de ambiente explícitas, compatíveis com a secção **Environment variables** do Portainer.

1. No Portainer, criar uma nova **Stack**
2. Colar o conteúdo do `docker-compose.yml`
3. Na secção **Environment variables**, preencher os valores
4. Clicar em **Deploy the stack**

---

## Tecnologias utilizadas

| Componente | Tecnologia |
|-----------|-----------|
| Linguagem | [Go 1.25.9](https://go.dev/) |
| Framework HTTP | [Gin](https://gin-gonic.com/) |
| OpenAPI / Swagger | [Huma v2](https://huma.rocks/) — OAS 3.1 automático |
| Detecção de QR codes | [gozxing](https://github.com/makiuchi-d/gozxing) |
| Renderização de PDF | [poppler-utils](https://poppler.freedesktop.org/) (`pdftoppm`) |
| Interface web | HTML + Tailwind CSS + Vanilla JS |
| Containerização | Docker / Docker Compose / Traefik |

---

## Licença

Distribuído sob a licença **MIT**. Consulte o ficheiro [LICENSE](LICENSE) para mais informações.

---

## Autor

**Ricardo Grangeia** — [https://ricardo.grangeia.pt](https://ricardo.grangeia.pt)
