const newurlform = document.getElementById('newurlform');
const sitekey = document.getElementById('sitekey').value;
function onSubmit(e) {
    // check if the url is valid
    const url = document.getElementById('url').value;
    if (url === '') {
        alert('Please enter a valid URL');
        return;
    }
    // send the url to the server
    newurlform.submit();
}
