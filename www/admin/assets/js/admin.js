$(document).ready(function(){
  $("#input-search").on("keyup", function() {
    const value = $(this).val().toLowerCase();
    $("#job-table tbody tr").filter(function() {
      $(this).toggle($(this).text().toLowerCase().indexOf(value) > -1)
    });
  });
});

function openModal(pin, price, format) {
  $("#input-disabled-pin").val(pin);
  $("#input-disabled-price").val(price + "€");
  $("#input-format").val(format);

  $('#modal-print').modal('show');
}

function closeModal() {
  $('#modal-print').modal('hide');
}

function print() {
  const pin = $("#input-disabled-pin").val();
  const internal = $("#input-internal").val();
  const printer = $("#input-printer").val();
  const format = $("#input-format").val();

  let data = new FormData();
  data.append('pin', pin);
  data.append('internal', internal);
  data.append('printer', printer);
  data.append('format', format);

  alert("Printing: " + pin + " Internal " + internal + " on printer " + printer);

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
