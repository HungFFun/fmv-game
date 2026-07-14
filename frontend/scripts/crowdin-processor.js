#!/usr/bin/env node

// Flattens Crowdin export format → react-i18next-friendly flat key/value JSON.
//
// Input  : /crowdin-imports/{lang}.json  (array of { identifier, source_string, translation, ... })
// Output : /public/locales/{appLangCode}/common.json  ({ identifier: translation })
//
// Run via:  npm run crowdin:processor
//
// Nguồn của dự án là TIẾNG VIỆT; `en` (và các ngôn ngữ khác sau này) là bản dịch tải về.

const fs = require('fs').promises
const fsSync = require('fs')
const path = require('path')

// Maps the filename produced by Crowdin download → the locale folder in /public/locales/.
// Thêm dòng mới ở đây khi thêm ngôn ngữ (và đồng bộ supportedLanguages + LANGUAGES).
const LANGUAGE_MAPPING = {
    'vi.json': 'vi',
    'en.json': 'en',
}

const IMPORTS_DIR = path.resolve(__dirname, '../crowdin-imports')
const OUTPUT_DIR = path.resolve(__dirname, '../public/locales')

async function processFile(sourcePath, targetLang) {
    const raw = await fs.readFile(sourcePath, 'utf8')

    // Crowdin returns either an array of objects OR a JSON-lines stream.
    let items
    try {
        items = JSON.parse(raw)
        if (!Array.isArray(items)) items = [items]
    } catch {
        items = JSON.parse(`[${raw.trim().replace(/}\s*{/g, '},{')}]`)
    }

    const translations = {}
    let used = 0
    let skipped = 0

    for (const item of items) {
        if (!item || !item.identifier) {
            skipped++
            continue
        }
        const value = item.translation || item.source_string || ''
        if (!value) {
            skipped++
            continue
        }
        translations[item.identifier] = value
        used++
    }

    if (used === 0) {
        throw new Error(`No valid translations in ${path.basename(sourcePath)}`)
    }

    const outPath = path.join(OUTPUT_DIR, targetLang, 'common.json')
    await fs.mkdir(path.dirname(outPath), { recursive: true })
    await fs.writeFile(outPath, JSON.stringify(translations, null, 2), 'utf8')

    return { lang: targetLang, used, skipped }
}

async function main() {
    if (!fsSync.existsSync(IMPORTS_DIR)) {
        console.error(`❌ ${IMPORTS_DIR} not found. Download from Crowdin first.`)
        process.exit(1)
    }

    const files = (await fs.readdir(IMPORTS_DIR))
        .filter(f => f.endsWith('.json'))
        .map(f => ({ file: path.join(IMPORTS_DIR, f), name: f, lang: LANGUAGE_MAPPING[f] }))
        .filter(x => x.lang)

    if (files.length === 0) {
        console.error('❌ No mapped language files found in crowdin-imports/')
        process.exit(1)
    }

    console.log(`📁 Processing ${files.length} language file(s)…\n`)

    const results = await Promise.allSettled(
        files.map(({ file, lang }) => processFile(file, lang)),
    )

    const ok = []
    const failed = []
    results.forEach((r, i) => {
        const { name, lang } = files[i]
        if (r.status === 'fulfilled') {
            ok.push(r.value)
            console.log(`✅ ${lang.padEnd(6)} ${r.value.used} keys (${name})`)
        } else {
            failed.push({ lang, name, error: r.reason?.message })
            console.error(`❌ ${lang.padEnd(6)} ${r.reason?.message} (${name})`)
        }
    })

    console.log(`\nSummary: ${ok.length} ok, ${failed.length} failed`)

    // Clean up intermediate files only if ALL succeeded; keep on failure for debugging.
    if (failed.length === 0) {
        for (const { file } of files) {
            await fs.unlink(file).catch(() => {})
        }
        console.log('🧹 Removed processed import files.')
    } else {
        console.log('⚠️  Kept crowdin-imports/ for debugging.')
        process.exit(1)
    }
}

main().catch(err => {
    console.error('❌ Fatal:', err.message)
    process.exit(1)
})
