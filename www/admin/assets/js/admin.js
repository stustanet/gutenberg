(function() {
  const modal = document.getElementById("print-modal");

  const span = document.getElementsByClassName("modal-close")[0];

  span.onclick = closeModal;

  window.onclick = (event) => {
    if (event.target === modal) {
      closeModal();
    }
  };
})();

function openModal(pin, price, format) {
  document.getElementById("pin").innerText = pin;
  document.getElementById("price").innerText = price;
  document.getElementById("format").value = format;
  document.getElementById("print-modal").style.display = "block"
}

function closeModal() {
  document.getElementById("print-modal").style.display = "none"
}

function print() {
  const pin = document.getElementById("pin").innerText;
  const internal = document.getElementById("internal").checked;
  const printer = document.getElementById("printer").value;
  const format = document.getElementById("format").value;

  const data = new FormData();
  data.append('pin', pin);
  data.append('internal', internal);
  data.append('printer', printer);
  data.append('format', format);

  // alert("Printing: " + pin + " Internal " + internal + " on printer " + printer);

  const xhr = new XMLHttpRequest();
  xhr.open('POST', "/print", true);
  xhr.onload = function () {
    if (this.status === 200) {
      alert("Success: " + xhr.response);
    } else {
      alert("FAIL: " + xhr.response);
    }
  };
  xhr.send(data);

  closeModal();
}
