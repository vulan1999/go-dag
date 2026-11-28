(function () {
	function renderDAGEdges() {
		const canvas = document.getElementById("dag-canvas");
		const svg = document.getElementById("dag-edges");
		if (!canvas || !svg) return;

		let edges = [];
		try {
			edges = JSON.parse(canvas.dataset.edges || "[]");
		} catch (e) {
			return;
		}

		const canvasRect = canvas.getBoundingClientRect();
		svg.setAttribute("width", canvas.scrollWidth);
		svg.setAttribute("height", canvas.scrollHeight);
		svg.innerHTML = "";

		const defs = document.createElementNS("http://www.w3.org/2000/svg", "defs");
		const marker = document.createElementNS("http://www.w3.org/2000/svg", "marker");
		marker.setAttribute("id", "dag-arrow");
		marker.setAttribute("viewBox", "0 0 10 10");
		marker.setAttribute("refX", "9");
		marker.setAttribute("refY", "5");
		marker.setAttribute("markerWidth", "7");
		marker.setAttribute("markerHeight", "7");
		marker.setAttribute("orient", "auto-start-reverse");

		const arrowPath = document.createElementNS("http://www.w3.org/2000/svg", "path");
		arrowPath.setAttribute("d", "M 0 1 L 10 5 L 0 9 z");
		arrowPath.setAttribute("fill", "#6b7280");
		marker.appendChild(arrowPath);
		defs.appendChild(marker);
		svg.appendChild(defs);

		const nodeMap = new Map();
		canvas.querySelectorAll(".dag-node").forEach((el) => {
			nodeMap.set(el.dataset.nodeId, el);
		});

		edges.forEach((edge) => {
			const fromEl = nodeMap.get(edge.from);
			const toEl = nodeMap.get(edge.to);
			if (!fromEl || !toEl) return;

			const fromRect = fromEl.getBoundingClientRect();
			const toRect = toEl.getBoundingClientRect();

			const x1 = fromRect.right - canvasRect.left + canvas.scrollLeft;
			const y1 = fromRect.top + fromRect.height / 2 - canvasRect.top + canvas.scrollTop;
			const x2 = toRect.left - canvasRect.left + canvas.scrollLeft;
			const y2 = toRect.top + toRect.height / 2 - canvasRect.top + canvas.scrollTop;

			const dx = Math.max(Math.abs(x2 - x1) * 0.5, 40);
			const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
			path.setAttribute("d", `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`);
			path.setAttribute("fill", "none");
			path.setAttribute("stroke", "#6b7280");
			path.setAttribute("stroke-width", "2");
			path.setAttribute("marker-end", "url(#dag-arrow)");
			svg.appendChild(path);
		});
	}

	document.addEventListener("DOMContentLoaded", renderDAGEdges);
	window.addEventListener("resize", renderDAGEdges);
	document.body.addEventListener("htmx:afterSwap", renderDAGEdges);

	window.addMappingRow = function (btn, kind) {
		const section = btn.closest(".mapping-section");
		if (!section) return;
		const list = section.querySelector(".mapping-rows");
		if (!list) return;

		const row = document.createElement("div");
		row.className = "mapping-row";

		if (kind === "rename") {
			row.innerHTML =
				'<input type="text" name="rename_from[]" placeholder="Source column"/>' +
				'<span class="mapping-arrow">&rarr;</span>' +
				'<input type="text" name="rename_to[]" placeholder="Target column (optional)"/>' +
				'<button type="button" class="btn btn-sm btn-danger" onclick="this.closest(\'.mapping-row\').remove()">&times;</button>';
		} else {
			row.innerHTML =
				'<input type="text" name="set_key[]" placeholder="Field name"/>' +
				'<span class="mapping-arrow">=</span>' +
				'<input type="text" name="set_val[]" placeholder="Static value"/>' +
				'<button type="button" class="btn btn-sm btn-danger" onclick="this.closest(\'.mapping-row\').remove()">&times;</button>';
		}
		list.appendChild(row);
	};
})();
