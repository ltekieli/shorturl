shortenerForm.onsubmit = async (e) => {
    e.preventDefault();
    var form = document.querySelector("#shortenerForm");

    data = {
      url : form.querySelector('input[name="longid"]').value,
    }

    let response = await fetch('http://localhost:8090/api/shorten', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data),
    })


    let text = await response.text(); // read response body as text

    try {
        data = window.location.protocol + "//" + window.location.host + "/x/" + JSON.parse(text).url
    } catch (e) {
        data = "Error: " + text;
    }

    document.querySelector("#shortid").innerHTML = data;
};
