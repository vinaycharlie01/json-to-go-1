function toProperCase(str) {
    // ensure that the SCREAMING_SNAKE_CASE is converted to snake_case
    if (str.match(/^[_A-Z0-9]+$/)) {
        str = str.toLowerCase();
    }

    // https://github.com/golang/lint/blob/5614ed5bae6fb75893070bdc0996a68765fdd275/lint.go#L771-L810
    const commonInitialisms = [
        "ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP",
        "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA",
        "SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "UID", "UUID",
        "URI", "URL", "UTF8", "VM", "XML", "XMPP", "XSRF", "XSS"
    ];

    return str.replace(/(^|[^a-zA-Z])([a-z]+)/g, function (unused, sep, frag) {
        if (commonInitialisms.indexOf(frag.toUpperCase()) >= 0)
            return sep + frag.toUpperCase();
        else
            return sep + frag[0].toUpperCase() + frag.substr(1).toLowerCase();
    }).replace(/([A-Z])([a-z]+)/g, function (unused, sep, frag) {
        if (commonInitialisms.indexOf(sep + frag.toUpperCase()) >= 0)
            return (sep + frag).toUpperCase();
        else
            return sep + frag;
    });
}


// // Example usage
// const inputString = "EXAMPLE_STRING";  // Replace this with your input string
// const properCaseString = toProperCase(inputString);
// console.log(properCaseString);

const inputs = [
    "$SCREAMING_SNAKE_CASE",
    "camelCase",
    "kebab-case",
    "Title Case",
    "lowercase",
    "HTML",
    "API",
    "HTTP",
    "CSS",
    "JSON",
    "URL"
];

inputs.forEach(input => {
    const result = toProperCase(input);
    console.log(`Input: ${input} => Output: ${result}`);
});



