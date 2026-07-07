# Kross Licensing System

[![Go Reference](https://pkg.go.dev/badge/github.com/user/kross.svg)](https://pkg.go.dev/github.com/user/kross)
[![Go Report Card](https://goreportcard.com/badge/github.com/user/kross)](https://goreportcard.com/report/github.com/user/kross)

Kross is a secure, cross-platform software licensing system written in Go. It supports both offline verification (using Ed25519 signatures) and online activation, license background synchronization, local blacklisting, HWID-bound checks, and application obfuscation support.

---

## 📖 Документация / Documentation

Detailed guides are split by topic and available in Russian and English:

### 🇷🇺 Русский (Russian)
- [Использование CLI](docs/ru/cli.md) — генерация ключей, выпуск лицензий (персональных и массовых).
- [Настройка и запуск Сервера](docs/ru/server.md) — развертывание SQLite-сервера лицензий, логика отзыва ключей.
- [Интеграция в Приложение](docs/ru/integration.md) — использование клиентской библиотеки, вызов GUI-окна активации, фоновая синхронизация и обфускация с помощью `garble`.

### 🇺🇸 English
- [CLI Usage Guide](docs/en/cli.md) — key generation and issuing personal/mass licenses.
- [Server Deployment Guide](docs/en/server.md) — running the SQLite activation server and understanding the HTTP API.
- [Client Integration Guide](docs/en/integration.md) — integrating the client library, launching the Fyne GUI, background sync, and building obfuscated binaries with `garble`.

---

## ⚡ Quick Start / Быстрый старт

### 1. Build the project / Сборка проекта
Run the provided build script to compile the Server, CLI, and compile an obfuscated client example:
```bash
./scripts/build.sh
```

### 2. Generate a key pair / Генерация пары ключей
```bash
./dist/kross-cli keygen
```
This generates `private.key` and `public.key` in your current directory.

### 3. Start the Server / Запуск сервера
```bash
./dist/kross-server -addr=":8080" -db="kross.db"
```

### 4. Issue a License / Выпуск лицензии
Issue a personal license tied to an email:
```bash
./dist/kross-cli issue --email="user@example.com" --days=365
```

---

## 🛡️ License / Лицензия

This project is licensed under the MIT License - see the LICENSE file for details.
