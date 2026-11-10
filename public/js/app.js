
let items = [];

let claims = [
    {
        id: "CL-021",
        item: "Blue water bottle with sticker",
        person: "Student 221",
        proof: "Sticker and colour description matched",
        status: "Pending"
    },
    {
        id: "CL-018",
        item: "Introductory programming textbook",
        person: "Student 094",
        proof: "Name inside cover matched",
        status: "Claimed"
    }
];

// Prevent database text from being interpreted as HTML.
function escapeHTML(value) {
    return String(value ?? "").replace(/[&<>"']/g, char => ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        '"': "&quot;",
        "'": "&#39;"
    })[char]);
}

// Load actual items from MySQL through the Go API.
async function loadItems() {
    try {
        const response = await fetch("/api/items");

        if (!response.ok) {
            throw new Error("Could not retrieve items.");
        }

        const data = await response.json();

        items = data.map(item => ({
            id: item.id,
            desc: item.description,
            cat: item.category,
            date: item.dateFound,
            loc: item.location,
            finder: "",
            status: item.status === "Pending Verification"
                ? "Pending"
                : item.status,
            photo: item.photo
        }));

        render();

    } catch (error) {
        console.error("Database search error:", error);
        results.innerHTML = "<p>Unable to load items. Please try again.</p>";
    }
}

// Navigation
document.querySelectorAll("nav button").forEach(button => {
    button.onclick = () => {
        document.querySelectorAll("nav button, .page")
            .forEach(element => element.classList.remove("active"));

        button.classList.add("active");

        document.getElementById(button.dataset.id)
            .classList.add("active");
    };
});

// Status labels
function tag(status) {
    return `<span class="tag ${
        status === "Claimed" ? "claimed" :
        status === "Pending" ? "pending" : ""
    }">${escapeHTML(status)}</span>`;
}

// Render dashboard and search results
function render() {
    recent.innerHTML = items.slice(-5).reverse().map(item => `
        <tr>
            <td>${escapeHTML(item.desc)}</td>
            <td>${escapeHTML(item.cat)}</td>
            <td>${escapeHTML(item.date)}</td>
            <td>${escapeHTML(item.loc)}</td>
            <td>${tag(item.status)}</td>
        </tr>
    `).join("");

    drawSearch();
    stats();
}

// Public item search
function drawSearch() {
    let term = q.value.toLowerCase();

    let filteredItems = items.filter(item =>
        item.desc.toLowerCase().includes(term) &&
        (!fc.value || item.cat === fc.value) &&
        (!fd.value || item.date === fd.value) &&
        (!fl.value || item.loc === fl.value)
    );

    results.innerHTML = filteredItems.length
        ? filteredItems.map(item => `
            <div class="item">
                <div class="photo">⌕</div>

                <div>
                    <b>${escapeHTML(item.desc)}</b>

                    <p>
                        ${escapeHTML(item.cat)} · ${escapeHTML(item.date)}
                        <br>
                        Found at: ${escapeHTML(item.loc)}
                    </p>

                    ${tag(item.status)}

                    <p>
                        Visit the central office with details
                        that can verify ownership.
                    </p>
                </div>
            </div>
        `).join("")
        : "<p>No matching items found.</p>";
}

// Search filters
[q, fc, fd, fl].forEach(input => {
    input.addEventListener("input", drawSearch);
    input.addEventListener("change", drawSearch);
});

// Dashboard statistics
async function stats() {
    try {
        const response = await fetch("/api/reports", {
            credentials: "same-origin"
        });

        if (!response.ok) {
            throw new Error("Could not load report statistics.");
        }

        const data = await response.json();

        // Update report totals
        document.getElementById("rtotal").textContent = data.total;
        document.getElementById("rclaimed").textContent = data.claimed;
        document.getElementById("rpending").textContent = data.pending;
        document.getElementById("rate").textContent =
            Math.round(data.rate) + "%";

        // Update category summary
        const totalItems = data.total;

        document.getElementById("bars").innerHTML =
            data.categories.map(category => `
                <p>
                    ${escapeHTML(category.category)}:
                    <b>${category.count}</b> / ${totalItems}
                </p>
            `).join("");

    } catch (error) {
        console.error("Statistics error:", error);

        document.getElementById("bars").textContent =
            "Unable to load statistics. Please log in as staff.";
    }
}



// Claims

async function claimRender() {
    const claimRows = document.getElementById("claimrows");

    claimRows.innerHTML = `
        <tr>
            <td colspan="6">Loading claims...</td>
        </tr>
    `;

    try {
        const response = await fetch("/api/claims", {
            credentials: "same-origin"
        });

        if (!response.ok) {
            throw new Error("Could not retrieve claims.");
        }

        claims = await response.json();

        if (claims.length === 0) {
            claimRows.innerHTML = `
                <tr>
                    <td colspan="6">No claims have been submitted yet.</td>
                </tr>
            `;
            return;
        }

        claimRows.innerHTML = claims.map((claim, index) => `
            <tr>
                <td>${escapeHTML(claim.id)}</td>
                <td>${escapeHTML(claim.item)}</td>
                <td>${escapeHTML(claim.person)}</td>
                <td>${escapeHTML(claim.proof)}</td>
                <td>${tag(claim.status)}</td>
                <td>
                    ${claim.status === "Pending"
                        ? `<button class="primary" onclick="decision(${index})">Review claim</button>`
                        : "—"}
                </td>
            </tr>
        `).join("");

    } catch (error) {
        console.error(error);
        claimRows.innerHTML = `
            <tr>
                <td colspan="6">Unable to load claims. Please try again.</td>
            </tr>
        `;
    }
}

async function decision(index) {
    const claim = claims[index];

    if (!claim || claim.status !== "Pending") {
        return;
    }

    const choice = prompt(
        "Enter Approved or Rejected for this claim:"
    );

    if (!choice) return;

    const decisionValue = choice.trim().toLowerCase();

    if (decisionValue !== "approved" && decisionValue !== "rejected") {
        alert("Please enter Approved or Rejected.");
        return;
    }

    const notes = prompt("Enter verification notes:");

    if (!notes || !notes.trim()) {
        alert("Verification notes are required.");
        return;
    }

    try {
        const response = await fetch("/api/claims/decision", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            credentials: "same-origin",
            body: JSON.stringify({
                claimId: claim.id,
                decision: decisionValue === "approved"
                    ? "Approved"
                    : "Rejected",
                notes: notes.trim()
            })
        });

        const data = await response.json();

        if (!response.ok) {
            alert(data.error || "Could not save the decision.");
            return;
        }

        alert("Claim decision saved successfully!");

        await claimRender();
        await loadItems();

    } catch (error) {
        console.error(error);
        alert("Unable to connect to the server.");
    }
}


// Design specifications
let descriptions = [
    [
        "Staff Login Form",
        "Authorized staff (Ms. Amanda and office staff).",
        "Authenticate staff before protected functions are available.",
        "Client: UB Lost & Found web application. Environment: desktop web browser on an office computer."
    ],
    [
        "Add Found Item Form",
        "Office staff.",
        "Record description, category, location, date found, finder information, and photo.",
        "Client: UB Lost & Found web application. Environment: desktop browser at the central office."
    ],
    [
        "Public Item Search Page",
        "Students, faculty, and staff.",
        "Search the public inventory by keyword, category, date range, and location; view item details and claiming instructions.",
        "Client: UB Lost & Found web application. Environment: desktop web browser; no account required for public search."
    ],
    [
        "Claim Verification & Retrieval Form",
        "Office staff assisting claimants.",
        "Compare proof of ownership with the item record, record verification notes, and update status to Claimed or Pending.",
        "Client: UB Lost & Found web application. Environment: desktop browser at the central office; some checks take place in person."
    ],
    [
        "Inventory Statistics Report",
        "Authorized office staff / administrators.",
        "Review item totals, claim outcomes, and category summaries.",
        "Client: UB Lost & Found web application. Environment: desktop browser with staff access."
    ]
];

specgrid.innerHTML = descriptions.map(description => `
    <div class="panel">
        <b>${description[0]}</b>

        <p>
            <b>(a) Narrative Overview</b><br>
            Target users: ${description[1]}<br>
            Task: ${description[2]}<br>
            ${description[3]}
        </p>

        <p>
            <b>(b) Sample Design</b><br>
            Use the matching screen in this prototype as the desktop mockup screenshot.
        </p>

        <p>
            <b>(c) Testing & Usability</b><br>
            Collect four actual user scores for consistency, sufficiency, and accuracy; enter them in the score sheet.
        </p>
    </div>
`).join("");

// Usability score averages
document.querySelectorAll(".score").forEach(input => {
    input.oninput = () => {
        let row = input.dataset.r;

        let values = [
            ...document.querySelectorAll(`.score[data-r="${row}"]`)
        ]
            .map(element => Number(element.value))
            .filter(value => value >= 1 && value <= 7);

        document.getElementById("avg" + row).textContent =
            values.length
                ? (values.reduce((a, b) => a + b, 0) / values.length).toFixed(2)
                : "—";
    };
});

// Initial loading
claimRender();
loadItems();
// Staff Login
const loginForm = document.getElementById("loginForm");

if (loginForm) {
    loginForm.addEventListener("submit", async function (event) {
        event.preventDefault();

        const email = document.getElementById("staffEmail").value;
        const password = document.getElementById("staffPassword").value;
        const message = document.getElementById("loginMsg");

        message.textContent = "Checking login...";

        try {
            const response = await fetch("/api/login", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json"
                },
                credentials: "same-origin",
                body: JSON.stringify({ email, password })
            });

            const data = await response.json();

            if (!response.ok) {
                message.textContent = data.error || "Login failed.";
                return;
            }

            message.textContent = "Welcome, " + data.name + "!";
            loginForm.reset();

        } catch (error) {
            message.textContent = "Unable to connect to the server.";
        }
    });
}

// Save Found Item to MySQL
const databaseItemForm = document.getElementById("itemForm");

if (databaseItemForm) {
    databaseItemForm.addEventListener("submit", async function (event) {
        event.preventDefault();

        const message = document.getElementById("formmsg");

        const itemData = {
            description: document.getElementById("desc").value,
            category: document.getElementById("cat").value,
            location: document.getElementById("loc").value,
            dateFound: document.getElementById("date").value,
            finderInfo: document.getElementById("finder").value
        };
        console.log("ITEM FORM DATA:", itemData);

        message.textContent = "Saving item...";

        try {
            const response = await fetch("/api/items", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json"
                },
                credentials: "same-origin",
                body: JSON.stringify(itemData)
            });

            const data = await response.json();

            if (!response.ok) {
                message.textContent = data.error || "Could not save item.";
                return;
            }

            message.textContent = "Item saved successfully!";
           databaseItemForm.reset();

            // Refresh the public item list.
            if (typeof loadItems === "function") {
                await loadItems();
            }

        } catch (error) {
            message.textContent = "Unable to connect to the server.";
        }
    });
}

// Staff logout
const logoutBtn = document.getElementById("logoutBtn");
const staffSession = document.getElementById("staffSession");
const staffWelcome = document.getElementById("staffWelcome");
const loginFormElement = document.getElementById("loginForm");

async function checkStaffSession() {
    try {
        const response = await fetch("/api/me", {
            credentials: "same-origin"
        });

        if (!response.ok) {
            staffSession.hidden = true;
            loginFormElement.hidden = false;
            return;
        }

        const staff = await response.json();

        staffWelcome.textContent =
            "Logged in as " + staff.name + " (" + staff.role + ")";

        staffSession.hidden = false;
        loginFormElement.hidden = true;

    } catch (error) {
        console.error("Could not check staff session:", error);
    }
}

if (logoutBtn) {
    logoutBtn.addEventListener("click", async function () {
        try {
            const response = await fetch("/api/logout", {
                method: "POST",
                credentials: "same-origin"
            });

            if (!response.ok) {
                alert("Could not log out. Please try again.");
                return;
            }

            staffSession.hidden = true;
            loginFormElement.hidden = false;
            loginFormElement.reset();

            document.getElementById("loginMsg").textContent =
                "You have successfully logged out.";

        } catch (error) {
            alert("Unable to connect to the server.");
        }
    });
}

checkStaffSession();

// Load available items into the public claim form
async function loadClaimItems() {
    const select = document.getElementById("claimItem");

    if (!select) return;

    try {
        const response = await fetch("/api/items");

        if (!response.ok) {
            throw new Error("Could not load items.");
        }

        const availableItems = await response.json();

        select.innerHTML = '<option value="">Select a found item</option>';

        availableItems
            .filter(item => item.status !== "Claimed")
            .forEach(item => {
                const option = document.createElement("option");

                option.value = item.id;
                option.textContent = item.description;

                select.appendChild(option);
            });

    } catch (error) {
        console.error("Claim items error:", error);
        select.innerHTML = '<option value="">Unable to load items</option>';
    }
}

// Submit a public claim
const publicClaimForm = document.getElementById("claimForm");

if (publicClaimForm) {
    publicClaimForm.addEventListener("submit", async function(event) {
        event.preventDefault();

        const message = document.getElementById("claimMessage");

        const claimData = {
            itemId: document.getElementById("claimItem").value,
            name: document.getElementById("claimName").value.trim(),
            type: document.getElementById("claimType").value,
            ubId: document.getElementById("claimID").value.trim(),
            contact: document.getElementById("claimContact").value.trim(),
            proof: document.getElementById("claimProof").value.trim()
        };

        message.textContent = "Submitting claim...";

        try {
            const response = await fetch("/api/claims/submit", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json"
                },
                body: JSON.stringify(claimData)
            });

            const data = await response.json();

            if (!response.ok) {
                message.textContent = data.error || "Could not submit claim.";
                return;
            }

            message.textContent = data.message;
            publicClaimForm.reset();

            await loadClaimItems();
            if (typeof loadItems === "function") {
                await loadItems();
            }

        } catch (error) {
            console.error("Claim submission error:", error);
            message.textContent = "Unable to connect to the server.";
        }
    });
}

loadClaimItems();
