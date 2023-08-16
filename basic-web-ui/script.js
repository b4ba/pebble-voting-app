addEventListener("DOMContentLoaded", function () {
    fetch("/api/pubkey").
        then((responce) => responce.text()).
        then((pubkey) => { document.getElementById("pubkey").innerText = pubkey; });
});

function showDialog(msg, btn) {
    let dialog = document.getElementById("dialog");
    dialog.getElementsByTagName("div")[0].innerText = msg;
    let buttons = dialog.getElementsByTagName("button");
    if (buttons.length > 0) {
        let button = buttons[0];
        if (btn) {
            button.style.display = "";
        } else {
            button.style.display = "none";
        }
    }
    dialog.style.display = "";
}


function closeDialog() {
    let dialog = document.getElementById("dialog");
    dialog.style.display = "none";
}

function switchPane(pane) {
    let main = document.getElementById("main");
    let election = document.getElementById("election");
    main.style.display = "none";
    election.style.display = "none";
    if (pane == "election")
        election.style.display = "";
    else
        main.style.display = "";
}

function joinElection() {
    let joinStr = document.getElementById("text-join").value;
    console.log(joinStr);
    showDialog("Joining election...", false);
    fetch("/api/election/join/" + encodeURIComponent(joinStr)).
        then((responce) => responce.text()).
        then(() => { showElection(joinStr); });
}

function showElection(invStr) {
    fetch("/api/election/info/" + invStr).
        then((response) => response.json()).
        then((data) => {
            document.getElementById("election-title").innerText = 'Election "' + data.title + '"';
            document.getElementById("election-description").innerText = data.description;
            closeDialog();
            switchPane("election");
        });
}