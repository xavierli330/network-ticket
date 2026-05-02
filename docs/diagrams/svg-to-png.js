const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

async function convert(file) {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  const filePath = path.resolve(file);
  await page.goto('file://' + filePath, { waitUntil: 'networkidle' });
  await page.waitForTimeout(2000);
  
  // Get SVG element
  const svg = await page.$('svg');
  if (!svg) {
    console.error('No SVG found in ' + file);
    await browser.close();
    return;
  }
  
  const box = await svg.boundingBox();
  await svg.screenshot({ 
    path: file.replace('.html', '.png'),
    clip: { x: 0, y: 0, width: box.width, height: box.height }
  });
  
  console.log('Converted: ' + file + ' -> ' + file.replace('.html', '.png'));
  await browser.close();
}

(async () => {
  const files = ['architecture-diagram.html', 'deployment-diagram.html', 'usage-flow-diagram.html'];
  for (const f of files) {
    await convert(f);
  }
})();
