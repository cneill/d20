document.addEventListener("DOMContentLoaded", function() {
    document.body.addEventListener("htmx:sseMessage", function(event) {
        var data = JSON.parse(event.detail.data);
        var history_div = document.getElementById("history")
        history_div.innerHTML = data.html;
    });
});
