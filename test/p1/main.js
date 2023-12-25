


// Adds a prefix to a number to make an appropriate identifier in Go
function formatNumber(str) {
    if (!str)
        return "";
    else if (str.match(/^\d+$/))
        str = "Num" + str;
    else if (str.charAt(0).match(/\d/)) {
        const numbers = {
            '0': "Zero_", '1': "One_", '2': "Two_", '3': "Three_",
            '4': "Four_", '5': "Five_", '6': "Six_", '7': "Seven_",
            '8': "Eight_", '9': "Nine_"
        };
        str = numbers[str.charAt(0)] + str.substr(1);
    }

    return str;
}



// Test the function
result = formatNumber("123")
console.log(result)

result = formatNumber("456")
console.log(result)

result = formatNumber("abc")
console.log(result)