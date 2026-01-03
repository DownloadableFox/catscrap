import * as d3 from "https://cdn.jsdelivr.net/npm/d3@7/+esm";

function transform_connections(connections) {
    const nodes = [];
    const nodeMap = new Map();
    const links = [];

    // Ensure we don't duplicate nodes if they appear in values but not keys
    for (const [character, neighbours] of Object.entries(connections)) {
        if (!nodeMap.has(character)) {
            const newNode = { id: character };
            nodeMap.set(character, newNode);
            nodes.push(newNode);
        }
        for (const neighbour of neighbours) {
            if (!nodeMap.has(neighbour)) {
                const newNode = { id: neighbour };
                nodeMap.set(neighbour, newNode);
                nodes.push(newNode);
            }
            links.push({ source: character, target: neighbour });
        }
    }
    return [nodes, links];
}

async function setup(connections) {
    const [nodes, links] = transform_connections(connections);

    const width = window.innerWidth;
    const height = window.innerHeight;

    const canvas = d3.select("body").append("canvas")
        .attr("width", width)
        .attr("height", height)
        .node();
    const context = canvas.getContext("2d");

    const img = new Image();
    img.src = "placeholder.jpg";
    await img.decode(); 

    let transform = d3.zoomIdentity;

    const simulation = d3.forceSimulation(nodes)
        .force("link", d3.forceLink(links).id(d => d.id).distance(150))
        .force("charge", d3.forceManyBody().strength(-300))
        .force("center", d3.forceCenter(width / 2, height / 2));

    for (let i = 0; i < 50; ++i) simulation.tick();

    d3.select(canvas).call(d3.zoom()
        .scaleExtent([0.05, 8])
        .on("zoom", (event) => {
            transform = event.transform;
            render();
        }));

    simulation.on("tick", render);

    function render() {
        context.save();
        context.clearRect(0, 0, width, height);
        context.translate(transform.x, transform.y);
        context.scale(transform.k, transform.k);

        // Draw Links with Arrows
        context.strokeStyle = "#999";
        context.lineWidth = 1;
        links.forEach(d => {
            drawArrowLine(context, d.source.x, d.source.y, d.target.x, d.target.y, 25);
        });

        // Draw Nodes
        nodes.forEach(d => {
            // Only draw detailed images/text if zoomed in enough (Performance optimization)
            if (transform.k > 0.4) {
                drawNode(context, d, img);
            } else {
                // Just draw a dot when zoomed far out
                context.beginPath();
                context.arc(d.x, d.y, 4, 0, 2 * Math.PI);
                context.fillStyle = "#999";
                context.fill();
            }
        });

        context.restore();
    }

    function drawNode(ctx, d, image) {
        const r = 20;
        ctx.save();
        // Draw Circular Clipping for Image
        ctx.beginPath();
        ctx.arc(d.x, d.y, r, 0, Math.PI * 2);
        ctx.clip();
        ctx.drawImage(image, d.x - r, d.y - r, r * 2, r * 2);
        ctx.restore();

        // Draw Label
        ctx.fillStyle = "white";
        ctx.font = "12px Arial";
        ctx.fillText(d.id, d.x + r + 5, d.y + 5);
    }

    function drawArrowLine(ctx, x1, y1, x2, y2, offset) {
        const headLength = 10;
        const dx = x2 - x1;
        const dy = y2 - y1;
        const angle = Math.atan2(dy, dx);
        
        // Shorten the line so it doesn't overlap the character image
        const startX = x1 + Math.cos(angle) * offset;
        const startY = y1 + Math.sin(angle) * offset;
        const endX = x2 - Math.cos(angle) * offset;
        const endY = y2 - Math.sin(angle) * offset;

        ctx.beginPath();
        ctx.moveTo(startX, startY);
        ctx.lineTo(endX, endY);
        ctx.stroke();

        // Draw Arrow Head
        ctx.beginPath();
        ctx.moveTo(endX, endY);
        ctx.lineTo(endX - headLength * Math.cos(angle - Math.PI / 6), endY - headLength * Math.sin(angle - Math.PI / 6));
        ctx.moveTo(endX, endY);
        ctx.lineTo(endX - headLength * Math.cos(angle + Math.PI / 6), endY - headLength * Math.sin(angle + Math.PI / 6));
        ctx.stroke();
    }
}

fetch("connections.json")
    .then(res => res.json())
    .then(connections => setup(connections));