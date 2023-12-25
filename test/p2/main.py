import re

class YourClassName:
    def to_proper_case2(self, s):
        # Ensure that the SCREAMING_SNAKE_CASE is converted to snake_case
        if re.match("^[_A-Z0-9]+$", s):
            s = s.lower()

        # List of common initialisms
        common_initialisms = {
            "ACL", "API", "ASCII", "CPU", "CSS", "DNS",
            "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID",
            "IP", "JSON", "LHS", "QPS", "RAM", "RHS",
            "RPC", "SLA", "SMTP", "SQL", "SSH", "TCP",
            "TLS", "TTL", "UDP", "UI", "UID", "UUID",
            "URI", "URL", "UTF8", "VM", "XML", "XMPP",
            "XSRF", "XSS",
        }

        # Convert the string to Proper Case
        s = re.sub(r'(^|[^a-zA-Z])([a-z]+)', lambda match: match.group(1) + match.group(2).upper() if match.group(2).upper() in common_initialisms else match.group(1) + match.group(2).capitalize(), s)

        s = re.sub(r'([A-Z])([a-z]+)', lambda match: match.group(1) + match.group(2) if match.group(1) + match.group(2).upper() in common_initialisms else match.group(1) + match.group(2), s)
        s = re.sub(r'[^a-zA-Z0-9_]', '', s)
        s = s.replace('_', '')

        return s
    def to_proper_case(self, s):
        # Ensure that the SCREAMING_SNAKE_CASE is converted to snake_case
        if re.match("^[_A-Z0-9]+$", s):
            s = s.lower()

        # List of common initialisms
        common_initialisms = {
            "ACL", "API", "ASCII", "CPU", "CSS", "DNS",
            "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID",
            "IP", "JSON", "LHS", "QPS", "RAM", "RHS",
            "RPC", "SLA", "SMTP", "SQL", "SSH", "TCP",
            "TLS", "TTL", "UDP", "UI", "UID", "UUID",
            "URI", "URL", "UTF8", "VM", "XML", "XMPP",
            "XSRF", "XSS",
        }

        # Convert the string to Proper Case
        s = re.sub(r'([a-zA-Z])_([a-zA-Z])', r'\1\2', s)  # Remove underscore between letters
        s = re.sub(r'(^|[^a-zA-Z_])([a-z_]+)', lambda match: match.group(1) + match.group(2).upper() if match.group(2).upper() in common_initialisms else match.group(1) + match.group(2).capitalize(), s)
        s = re.sub(r'([A-Z])([a-z_]+)', lambda match: match.group(1) + match.group(2) if match.group(1) + match.group(2).upper() in common_initialisms else match.group(1) + match.group(2), s)

        # Remove other special characters
        s = re.sub(r'[^a-zA-Z0-9_]', '', s)

        return s


# Example usage:
your_instance = YourClassName()
result = your_instance.to_proper_case2("_id")
print(result)
