// Check if an URL exists
function checkUrl(url) {
    var http = new XMLHttpRequest();
    http.open('HEAD', url, false);
    http.send();
    return http.status == 200;
}


// Get current browsed page
function getCurrentUrlPage(url) {
    var page = document.URL.split('/')[3]
    return page;
}


// Shortcut to get element
function element(element) {
    return document.getElementById(element);
}


// Display a notification notification
function displayError(message) {
    let notification = document.createElement('div');

    // Set the ID to always replace the same element with the notification
    notification.setAttribute("id", "error");
    notification.innerHTML = message;
    notification.className = 'notification notification-error';

    document.body.appendChild(notification);

    // Automatically delete notification after few seconds
    setTimeout(function () {
        document.body.removeChild(notification);
    }, 3000);
}


// Display an notification notification
function displayInfo(message) {
    let notification = document.createElement('div');

    // Set the ID to always replace the same element with the notification
    notification.setAttribute("id", "info");
    notification.innerHTML = message;
    notification.className = 'notification notification-info';

    document.body.appendChild(notification);

    // Automatically delete notification after few seconds
    setTimeout(function () {
        document.body.removeChild(notification);
    }, 3000);
}


// Convert date locale to UTC
function timeLocalToUtc(dateLocalToConvert) {
    const dateLocal = new Date(dateLocalToConvert);
    const dateUtc = new Date(dateLocal.getTime() - dateLocal.getTimezoneOffset());
    const dateUtcFormatted = dateUtc.toISOString().slice(0, 16).toString();

    return dateUtcFormatted
}


// Copy content of a div (from its ID) to clipboard
function copyToClipboard(div) {
    let content = document.getElementById(div).innerText;

    // Replace unwanted HTML characters
    content = content.replace(/\u00A0/g, ' ');

    navigator.clipboard.writeText(content)
        .then(() => displayInfo("Copied!"))
        .catch(err => displayError("Failed to copy"));
}


// Copy QRCode
async function copyToClipboardQRCode(elementId) {
  const container = document.getElementById(elementId);
  const canvas = container.querySelector("canvas");

  canvas.toBlob(async function (blob) {
    try {
      await navigator.clipboard.write([
        new ClipboardItem({
          "image/png": blob
        })
      ]);

      displayInfo("Copied!")
    } catch (err) {
      displayError("Failed to copy")
    }
  });
}


// Generate a QRcode with a given text
function generateQR(div, textDiv) {
    const container = document.getElementById(div);
    const text = document.getElementById(textDiv).innerText;
    console.log(container)
    console.log(text)

    new QRCode(container, {
        text: text,
        width: 200,
        height: 200,
        correctLevel: QRCode.CorrectLevel.H
    });
}