document.addEventListener("DOMContentLoaded", function() {
    document.body.addEventListener("htmx:sseMessage", function(event) {
        console.debug(event);
        var data = JSON.parse(event.detail.data);

        switch (event.detail.type) {
        case "ROLL":
            var history_div = document.getElementById("history")
            history_div.innerHTML = data.html;
            break;
        case "STATS":
            var stats_div = document.getElementById("stats")
            stats_div.outerHTML = data.html;
            break;
        }
    });
});
