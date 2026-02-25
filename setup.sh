#!/bin/bash
# setup-hackathon.sh

echo "🚀 Scaffolding Local-First Gig Network..."

# 1. Create Monorepo
cd skillzone
echo "node_modules/\n.env\n*.db\nbin/" > .gitignore

# 2. Setup Go Backend
mkdir backend && cd backend
go mod init github.com/carsonak/skillzone/backend
go get github.com/mattn/go-sqlite3 # SQLite driver
mkdir api models db
touch main.go
cd ..

# 3. Setup Vite PWA Frontend
npm create vite@latest frontend -- --template react
cd frontend
npm install
npm i dexie html5-qrcode # For IndexedDB and QR scanning
npm i -D vite-plugin-pwa tailwindcss postcss autoprefixer
npx tailwindcss init -p

echo "✅ Done!, start your engines."