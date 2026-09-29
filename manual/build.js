/**
 * @file build.js
 * @description BOMIX 使用說明書 PDF / HTML 自動建置腳本
 * 支援 Puppeteer 與 Windows 內建 Edge/Chrome 雙引擎轉換，零外部環境負擔。
 */

const fs = require('fs');
const path = require('path');
const { execFileSync, execSync } = require('child_process');

// 路徑定義
const ROOT_DIR = __dirname;
const MANUAL_MD_PATH = path.join(ROOT_DIR, 'manual.md');
const CSS_PATH = path.join(ROOT_DIR, 'styles', 'manual-pdf.css');
const IMAGES_DIR = path.join(ROOT_DIR, 'images');
const DIST_DIR = path.join(ROOT_DIR, 'dist');
const OUTPUT_HTML = path.join(DIST_DIR, 'BOMIX-使用者手冊.html');
const OUTPUT_PDF = path.join(DIST_DIR, 'BOMIX-使用者手冊.pdf');

// 確保輸出目錄存在
if (!fs.existsSync(DIST_DIR)) {
  fs.mkdirSync(DIST_DIR, { recursive: true });
}

/**
 * 簡易而強健的 Markdown 轉 HTML 解析器 (含表格、代碼、標題、清單)
 * 若有安裝 marked 則優先使用 marked，無安裝則使用內建轉換器
 */
function markdownToHtml(mdText) {
  try {
    const { marked } = require('marked');
    // 自訂 marked 選項
    marked.setOptions({
      gfm: true,
      breaks: true
    });
    return marked.parse(mdText);
  } catch (err) {
    console.log('[Build] 提示: 未安裝 marked 依賴，使用內建輕量級 Markdown 解析器...');
    return fallbackMarkdownParse(mdText);
  }
}

/**
 * 內建輕量 Markdown 解析器 (零依賴 Fallback，支援標題 h1-h6、表格、狀態機清單與有序序號)
 */
function fallbackMarkdownParse(md) {
  let lines = md.split(/\r?\n/);
  let html = [];
  let inTable = false;
  let tableHeader = true;
  let listStack = []; // 追蹤開啟的清單標籤: 'ol', 'li', 'ul'

  function closeLists() {
    while (listStack.length > 0) {
      const tag = listStack.pop();
      html.push(`</${tag}>`);
    }
  }

  for (let i = 0; i < lines.length; i++) {
    let line = lines[i];
    let trimmed = line.trim();

    // 1. 空行
    if (trimmed.length === 0) {
      if (inTable) {
        html.push('</tbody></table>');
        inTable = false;
      }
      continue;
    }

    // 2. HTML 直通 (包含 <div>, <p>, <!-- 等原生標籤)
    if (trimmed.startsWith('<') || trimmed.startsWith('</') || trimmed.startsWith('<!--')) {
      if (inTable) { html.push('</tbody></table>'); inTable = false; }
      closeLists();
      html.push(line);
      continue;
    }

    // 3. 表格處理
    if (trimmed.startsWith('|') && trimmed.endsWith('|')) {
      closeLists();
      let cells = line.split('|').map(c => c.trim()).filter((c, idx, arr) => idx > 0 && idx < arr.length - 1);
      
      // 表頭分隔線
      if (cells.every(c => /^:?-+:?$/.test(c))) {
        tableHeader = false;
        continue;
      }

      if (!inTable) {
        html.push('<table>');
        html.push('<thead><tr>');
        cells.forEach(c => html.push(`<th>${formatInline(c)}</th>`));
        html.push('</tr></thead><tbody>');
        inTable = true;
      } else {
        html.push('<tr>');
        cells.forEach(c => html.push(`<td>${formatInline(c)}</td>`));
        html.push('</tr>');
      }
      continue;
    } else if (inTable) {
      html.push('</tbody></table>');
      inTable = false;
    }

    // 4. 水平分隔線
    if (trimmed === '---' || trimmed === '***') {
      closeLists();
      html.push('<hr />');
      continue;
    }

    // 5. 標題 (優先判斷長前綴，避免 #### 被 ### 誤判)
    if (line.startsWith('###### ')) {
      closeLists();
      html.push(`<h6>${formatInline(line.substring(7))}</h6>`);
      continue;
    } else if (line.startsWith('##### ')) {
      closeLists();
      html.push(`<h5>${formatInline(line.substring(6))}</h5>`);
      continue;
    } else if (line.startsWith('#### ')) {
      closeLists();
      html.push(`<h4>${formatInline(line.substring(5))}</h4>`);
      continue;
    } else if (line.startsWith('### ')) {
      closeLists();
      html.push(`<h3>${formatInline(line.substring(4))}</h3>`);
      continue;
    } else if (line.startsWith('## ')) {
      closeLists();
      html.push(`<h2>${formatInline(line.substring(3))}</h2>`);
      continue;
    } else if (line.startsWith('# ')) {
      closeLists();
      html.push(`<h1>${formatInline(line.substring(2))}</h1>`);
      continue;
    }

    // 6. 有序清單項目 (例如 "1. 項目說明", "  2. 項目說明")
    let olMatch = trimmed.match(/^(\d+)\.\s+(.*)$/);
    if (olMatch) {
      let num = olMatch[1];
      let content = olMatch[2];

      // 若目前在子無序清單中，先關閉 ul
      if (listStack.length > 0 && listStack[listStack.length - 1] === 'ul') {
        html.push('</ul>');
        listStack.pop();
      }

      // 若上一層是 li，先關閉上一個 li
      if (listStack.length > 0 && listStack[listStack.length - 1] === 'li') {
        html.push('</li>');
        listStack.pop();
      }

      // 若目前不在 ol 中，開啟 ol
      if (listStack.length === 0 || listStack[listStack.length - 1] !== 'ol') {
        closeLists();
        html.push('<ol>');
        listStack.push('ol');
      }

      // 開啟帶有精確 value 的 li
      html.push(`<li value="${num}">${formatInline(content)}`);
      listStack.push('li');
      continue;
    }

    // 7. 無序清單項目 (例如 "- 項目說明", "  * 項目說明")
    let ulMatch = trimmed.match(/^[-*]\s+(.*)$/);
    if (ulMatch) {
      let content = ulMatch[1];

      // 若目前在 li 中 (表示這是有序項目的子清單)
      if (listStack.length > 0 && listStack[listStack.length - 1] === 'li') {
        html.push('<ul>');
        listStack.push('ul');
      } else if (listStack.length === 0 || listStack[listStack.length - 1] !== 'ul') {
        // 獨立的無序清單
        closeLists();
        html.push('<ul>');
        listStack.push('ul');
      }

      html.push(`<li>${formatInline(content)}</li>`);
      continue;
    }

    // 8. 一般段落文字
    closeLists();
    html.push(`<p>${formatInline(line)}</p>`);
  }

  if (inTable) html.push('</tbody></table>');
  closeLists();
  return html.join('\n');
}

/**
 * 行內語法格式化
 */
function formatInline(str) {
  return str
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.*?)\*/g, '<em>$1</em>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1" />');
}

/**
 * 將圖片內嵌為 Base64 Data URI，確保 PDF 或獨立 HTML 跨裝置完全無路徑丟失問題
 */
function inlineImages(html) {
  return html.replace(/<img\s+([^>]*?)src=["']([^"']+)["']([^>]*?)>/gi, (match, before, src, after) => {
    // 忽略外鏈或已經是 data: 的圖片
    if (src.startsWith('http://') || src.startsWith('https://') || src.startsWith('data:')) {
      return match;
    }

    // 尋找本地圖片路徑
    let imgPath = path.resolve(ROOT_DIR, src);
    if (!fs.existsSync(imgPath)) {
      // 嘗試相對於 images 目錄
      imgPath = path.resolve(IMAGES_DIR, path.basename(src));
    }

    if (fs.existsSync(imgPath)) {
      const ext = path.extname(imgPath).toLowerCase();
      let mimeType = 'image/png';
      if (ext === '.svg') mimeType = 'image/svg+xml';
      else if (ext === '.jpg' || ext === '.jpeg') mimeType = 'image/jpeg';
      else if (ext === '.webp') mimeType = 'image/webp';

      const fileData = fs.readFileSync(imgPath);
      const base64Data = fileData.toString('base64');
      const dataUri = `data:${mimeType};base64,${base64Data}`;
      return `<img ${before}src="${dataUri}"${after}>`;
    } else {
      console.warn(`[Build] 警告: 找不到圖片檔案: ${src} (已保留原路徑)`);
      return match;
    }
  });
}

/**
 * 尋找本機可用的 Chromium 核心 (Edge 或 Chrome)
 */
function findSystemBrowser() {
  const possiblePaths = [
    // Windows 64-bit Edge (Windows 11 預裝)
    'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
    'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe',
    // Windows 64-bit Chrome
    'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
    'C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe',
    // Local AppData
    path.join(process.env.LOCALAPPDATA || '', 'Microsoft', 'Edge', 'Application', 'msedge.exe'),
    path.join(process.env.LOCALAPPDATA || '', 'Google', 'Chrome', 'Application', 'chrome.exe'),
  ];

  for (const p of possiblePaths) {
    if (p && fs.existsSync(p)) {
      return p;
    }
  }
  return null;
}

/**
 * 主執行流程
 */
async function build() {
  console.log('====================================================');
  console.log('  BOMIX 說明書建置系統 (Manual PDF Builder)');
  console.log('====================================================\n');

  // 1. 讀取 Markdown 與 CSS
  if (!fs.existsSync(MANUAL_MD_PATH)) {
    console.error(`[Build] 錯誤: 找不到說明書檔案: ${MANUAL_MD_PATH}`);
    process.exit(1);
  }
  const mdContent = fs.readFileSync(MANUAL_MD_PATH, 'utf-8');
  const cssContent = fs.existsSync(CSS_PATH) ? fs.readFileSync(CSS_PATH, 'utf-8') : '';

  // 2. 轉換 Markdown 為 HTML
  console.log('[Build] 正在解析 Markdown 結構...');
  let rawBodyHtml = markdownToHtml(mdContent);

  // 3. 內嵌圖片為 Base64
  console.log('[Build] 正在內嵌高解析度操作圖片 (Base64)...');
  const inlinedBodyHtml = inlineImages(rawBodyHtml);

  // 4. 組裝完整 HTML 模板
  const fullHtml = `<!DOCTYPE html>
<html lang="zh-TW">
<head>
  <meta charset="UTF-8">
  <title>BOMIX 使用者操作手冊</title>
  <style>
${cssContent}
  </style>
</head>
<body>
${inlinedBodyHtml}
</body>
</html>`;

  // 寫入 HTML 檔案
  fs.writeFileSync(OUTPUT_HTML, fullHtml, 'utf-8');
  console.log(`[Build] ✅ 成功生成預覽 HTML: ${OUTPUT_HTML}`);

  // 檢查是否只需生成 HTML
  if (process.argv.includes('--html-only')) {
    console.log('[Build] 參數包含 --html-only，已跳過 PDF 轉換。');
    return;
  }

  // 5. 將 HTML 轉為高解析度向量 PDF
  console.log('[Build] 正在將 HTML 轉換為出版級 A4 PDF 檔案...');

  // 策略 A: 嘗試使用 puppeteer
  let renderedPdf = false;
  try {
    const puppeteer = require('puppeteer');
    console.log('[Build] 使用 Puppeteer 引擎進行渲染...');
    const browser = await puppeteer.launch({
      headless: 'new',
      args: ['--no-sandbox', '--disable-setuid-sandbox']
    });
    const page = await browser.newPage();
    await page.setContent(fullHtml, { waitUntil: 'networkidle0' });
    await page.pdf({
      path: OUTPUT_PDF,
      format: 'A4',
      printBackground: true,
      margin: { top: '0px', bottom: '0px', left: '0px', right: '0px' }
    });
    await browser.close();
    renderedPdf = true;
  } catch (err) {
    // Puppeteer 未安裝或失敗，轉用策略 B
    console.log(`[Build] 提示: 未安裝 Puppeteer 或執行略過 (${err.message})，啟用本機系統瀏覽器轉換...`);
  }

  // 策略 B: 使用本機安裝的 Microsoft Edge / Chrome Headless 模式
  if (!renderedPdf) {
    const browserPath = findSystemBrowser();
    if (!browserPath) {
      console.warn('[Build] ⚠️ 未在系統中找到 Edge 或 Chrome，無法直接輸出 PDF。');
      console.log(`[Build] 💡 提示：您可以使用任何瀏覽器開啟 ${OUTPUT_HTML}，並選擇「列印」->「另存為 PDF」。`);
      return;
    }

    console.log(`[Build] 使用系統瀏覽器: ${browserPath}`);
    const htmlFileUrl = `file:///${OUTPUT_HTML.replace(/\\/g, '/')}`;
    const tempPdfPath = path.join(DIST_DIR, '__temp_manual.pdf');
    if (fs.existsSync(tempPdfPath)) {
      try { fs.unlinkSync(tempPdfPath); } catch (_) {}
    }
    
    // Windows Edge/Chrome headless 命令參數
    const cmdArgs = [
      '--headless',
      '--disable-gpu',
      '--no-pdf-header-footer',
      '--run-all-compositor-stages-before-draw',
      `--print-to-pdf="${tempPdfPath}"`,
      `"${htmlFileUrl}"`
    ];

    try {
      execSync(`"${browserPath}" ${cmdArgs.join(' ')}`, { stdio: 'pipe' });
      if (fs.existsSync(tempPdfPath)) {
        // 嘗試覆蓋原檔
        let finalPdfPath = OUTPUT_PDF;
        let isLocked = false;
        try {
          fs.copyFileSync(tempPdfPath, OUTPUT_PDF);
          fs.unlinkSync(tempPdfPath);
        } catch (copyErr) {
          isLocked = true;
          finalPdfPath = path.join(DIST_DIR, 'BOMIX-使用者手冊-最新.pdf');
          fs.copyFileSync(tempPdfPath, finalPdfPath);
          try { fs.unlinkSync(tempPdfPath); } catch (_) {}
        }

        const stats = fs.statSync(finalPdfPath);
        console.log(`\n====================================================`);
        console.log(`  🎉 說明書建置完成！`);
        console.log(`  檔案路徑: ${finalPdfPath}`);
        console.log(`  檔案大小: ${(stats.size / 1024).toFixed(1)} KB`);
        if (isLocked) {
          console.warn(`  ⚠️ 提示: 偵測到 "${path.basename(OUTPUT_PDF)}" 正在被檢視軟體開啟鎖定。`);
          console.warn(`  💡 已先將最新版本輸出為: "${path.basename(finalPdfPath)}"`);
          console.warn(`  💡 若需覆蓋原檔，請關閉 PDF 檢視器後再次執行建置。`);
        }
        console.log(`====================================================\n`);
        renderedPdf = true;
      }
    } catch (e) {
      console.error(`[Build] 執行瀏覽器列印發生錯誤: ${e.message}`);
    }
  }
}

// 執行建置
build().catch(err => {
  console.error('[Build] 建置發生未預期錯誤:', err);
  process.exit(1);
});
