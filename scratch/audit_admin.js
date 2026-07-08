const puppeteer = require('puppeteer');

(async () => {
    console.log("Starting browser...");
    const browser = await puppeteer.launch({ headless: "new" });
    const page = await browser.newPage();
    
    // Set a viewport so we don't get mobile views
    await page.setViewport({ width: 1280, height: 800 });

    page.on('console', msg => console.log('PAGE LOG:', msg.text()));
    page.on('pageerror', err => console.log('PAGE ERROR:', err.message));

    console.log("Navigating to login page...");
    await page.goto('http://localhost:5175/login', { waitUntil: 'networkidle2' });

    console.log("Filling login form...");
    await page.type('input[type="email"]', 'admin@alhudhud.sa');
    await page.type('input[type="password"]', 'Admin@12345678');
    
    console.log("Submitting login...");
    await Promise.all([
        page.waitForNavigation({ waitUntil: 'networkidle2' }),
        page.click('button[type="submit"]')
    ]);

    console.log("Logged in. Current URL:", page.url());

    // Wait for the sidebar to load
    await page.waitForSelector('aside nav button');
    
    const errors = [];
    
    // List of menu items to check
    const menuItems = [
        'overview', 'companies', 'servers', 'alerts', 'distribution',
        'signalpush', 'tenants', 'tariffs', 'finance', 'reports',
        'support', 'gateways', 'staff', 'roles', 'audit'
    ];

    for (const item of menuItems) {
        console.log(`\nNavigating to ${item}...`);
        
        try {
            // Click the menu item
            await page.evaluate((id) => {
                const buttons = Array.from(document.querySelectorAll('aside nav button'));
                const target = buttons.find(b => b.innerText.toLowerCase().includes(id.toLowerCase()));
                if (target) target.click();
            }, item);

            // Wait a bit for the page to render
            await new Promise(r => setTimeout(r, 1000));

            // Check if there is an Error Boundary or a clear error message
            const pageText = await page.evaluate(() => document.body.innerText);
            
            if (pageText.includes("Something went wrong")) {
                const errorDetails = await page.evaluate(() => {
                    const pre = document.querySelector('pre');
                    return pre ? pre.innerText : "Unknown error";
                });
                console.log(`❌ ERROR on ${item}:`, errorDetails);
                errors.push({ page: item, error: errorDetails });
                
                // Need to reload to clear the error boundary
                console.log("Reloading to clear error boundary...");
                await page.reload({ waitUntil: 'networkidle2' });
            } else {
                console.log(`✅ ${item} loaded successfully.`);
            }
        } catch (e) {
            console.log(`Failed to navigate/check ${item}:`, e.message);
        }
    }

    console.log("\n--- AUDIT COMPLETE ---");
    if (errors.length > 0) {
        console.log("Found errors on the following pages:");
        errors.forEach(e => console.log(`- ${e.page}: ${e.error}`));
    } else {
        console.log("No errors found!");
    }

    await browser.close();
})();
