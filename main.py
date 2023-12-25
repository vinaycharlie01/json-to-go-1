import json
import re
import random

def json_to_go(json_str, typename=None, flatten=True, example=False, all_omitempty=False):
    def format_str(s):
        s = format_number(s)
        sanitized = to_proper_case(s).replace(r'[^a-z0-9]', "")
        if not sanitized:
            return "NAMING_FAILED"
        return format_number(sanitized)

    def format_number(s):
        if not s:
            return ""
        elif s.isdigit():
            s = "Num" + s
        elif s[0].isdigit():
            numbers = {'0': "Zero_", '1': "One_", '2': "Two_", '3': "Three_", '4': "Four_",
                       '5': "Five_", '6': "Six_", '7': "Seven_", '8': "Eight_", '9': "Nine_"}
            s = numbers.get(s[0], '') + s[1:]
        return s

    def to_proper_case(s):
        common_initialisms = [
            "ACL", "API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP",
            "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA",
            "SMTP", "SQL", "SSH", "TCP", "TLS", "TTL", "UDP", "UI", "UID", "UUID",
            "URI", "URL", "UTF8", "VM", "XML", "XMPP", "XSRF", "XSS"
        ]
        if re.match(r'^[_A-Z0-9]+$', s):
            s = s.lower()
        return re.sub(r'(^|[^a-zA-Z])([a-z]+)', lambda match: match.group(0) if match.group(2).upper() in common_initialisms else match.group(1) + match.group(2)[0].upper() + match.group(2)[1:].lower(), s)

    def unique_type_name(name, seen):
        if name not in seen:
            return name
        i = 0
        while True:
            new_name = name + str(i)
            if new_name not in seen:
                return new_name
            i += 1

    def go_type(val):
        if val is None:
            return "any"
        elif isinstance(val, str):
            if re.match(r'^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?(\+\d\d:\d\d|Z)$', val):
                return "time.Time"
            else:
                return "string"
        elif isinstance(val, (int, float)):
            if val % 1 == 0:
                return "int" if -2147483648 < val < 2147483647 else "int64"
            else:
                return "float64"
        elif isinstance(val, bool):
            return "bool"
        elif isinstance(val, list):
            return "slice"
        elif isinstance(val, dict):
            return "struct"
        else:
            return "any"

    def most_specific_possible_go_type(typ1, typ2):
        if typ1.startswith("float") and typ2.startswith("int"):
            return typ1
        elif typ1.startswith("int") and typ2.startswith("float"):
            return typ2
        else:
            return "any"

    def parse_scope(scope, depth=0, seen=None, stack=None, parent='', go_list=None):
        if go_list is None:
            go_list = []

        if isinstance(scope, (list, tuple)):
            slice_type = None
            for item in scope:
                this_type = go_type(item)
                if slice_type is None:
                    slice_type = this_type
                elif slice_type != this_type:
                    slice_type = most_specific_possible_go_type(this_type, slice_type)
                    if slice_type == "any":
                        break

            slice_str = f"[]"
            if flatten and depth >= 2:
                appender(slice_str, go_list)
            else:
                append(slice_str, go_list)

            if slice_type == "struct":
                all_fields = {}
                for item in scope:
                    keys = item.keys()
                    for key in keys:
                        key_name = key
                        if key_name not in all_fields:
                            all_fields[key_name] = {"value": item[key_name], "count": 0}
                        else:
                            existing_value = all_fields[key_name]["value"]
                            current_value = item[key_name]
                            if compare_objects(existing_value, current_value):
                                comparison_result = compare_object_keys(
                                    list(current_value.keys()),
                                    list(existing_value.keys())
                                )
                                if not comparison_result:
                                    key_name = f"{key_name}_{uuidv4()}"
                                    all_fields[key_name] = {"value": current_value, "count": 0}

                        all_fields[key_name]["count"] += 1

                struct_keys = all_fields.keys()
                struct_dict = {}
                omitempty_dict = {}
                for struct_key in struct_keys:
                    elem = all_fields[struct_key]
                    struct_dict[struct_key] = elem["value"]
                    omitempty_dict[struct_key] = elem["count"] != len(scope)

                parse_struct(depth + 1, struct_dict, go_list, seen, stack, parent, omitempty_dict)
            elif slice_type == "slice":
                parse_scope(scope[0], depth, seen, stack, parent, go_list)
            else:
                if flatten and depth >= 2:
                    appender(slice_type or "any", go_list)
                else:
                    append(slice_type or "any", go_list)

        elif isinstance(scope, dict):
            if flatten:
                if depth >= 2:
                    appender(parent, go_list)
                else:
                    append(parent, go_list)
            parse_struct(depth + 1, scope, go_list, seen, stack, parent, omitempty=None)
        else:
            if flatten and depth >= 2:
                appender(go_type(scope), go_list)
            else:
                append(go_type(scope), go_list)

    def parse_struct(depth, scope, go_list=None, seen=None, stack=None, parent='', omitempty=None):
        if go_list is None:
            go_list = []

        stack.append("\n" if depth >= 2 else "")

        seen_type_names = []

        if flatten and depth >= 2:
            parent_type = f"type {parent}"
            scope_keys = format_scope_keys(scope.keys())

            if parent in seen and compare_object_keys(scope_keys, seen[parent]):
                stack.pop()
                return

            seen[parent] = scope_keys
            appender(f"{parent_type} struct {{\n", go_list)
            inner_tabs = 1

            keys = list(scope.keys())
            for key in keys:
                key_name = get_original_name(key)
                indenter(inner_tabs, go_list)
                type_name = unique_type_name(format_str(key_name), seen_type_names)
                seen_type_names.append(type_name)
                appender(f"{type_name} ", go_list)
                parent = type_name
                parse_scope(scope[key], depth, seen, stack, parent, go_list)
                appender(f' `json:"{key_name}', go_list)
                if all_omitempty or (omitempty and omitempty[key] is True):
                    appender(',omitempty', go_list)
                appender('"`\n', go_list)

            indenter(inner_tabs - 1, go_list)
            appender("}", go_list)
        else:
            append("struct {\n", go_list)
            tabs = 1
            keys = list(scope.keys())
            for key in keys:
                key_name = get_original_name(key)
                indent(tabs, go_list)
                type_name = unique_type_name(format_str(key_name), seen_type_names)
                seen_type_names.append(type_name)
                append(f"{type_name} ", go_list)
                parent = type_name
                parse_scope(scope[key], depth, seen, stack, parent, go_list)
                append(f' `json:"{key_name}', go_list)
                if all_omitempty or (omitempty and omitempty[key] is True):
                    append(',omitempty', go_list)
                if example and scope[key] != "" and not isinstance(scope[key], (list, dict)):
                    append(f'" example:"{scope[key]}', go_list)
                append('"`\n', go_list)

            indent(tabs - 1, go_list)
            append("}", go_list)

        if flatten:
            go_list.append(stack.pop())

    def indent(tabs, go_list=None):
        if go_list is None:
            go_list = []
        for _ in range(tabs):
            go_list.append('\t')

    def append(s, go_list=None):
        if go_list is None:
            go_list = []
        go_list.append(s)

    def indenter(tabs, go_list=None):
        if go_list is None:
            go_list = []
        for _ in range(tabs):
            stack[-1] += '\t'

    def appender(s, go_list=None):
        if go_list is None:
            go_list = []
        stack[-1] += s

    def unique_type_name(name, seen):
        if name not in seen:
            return name
        i = 0
        while True:
            new_name = name + str(i)
            if new_name not in seen:
                return new_name
            i += 1

    def format_scope_keys(keys):
        return [format_str(key) for key in keys]

    def compare_objects(object_a, object_b):
        return isinstance(object_a, dict) and isinstance(object_b, dict)

    def compare_object_keys(item_a_keys, item_b_keys):
        length_a = len(item_a_keys)
        length_b = len(item_b_keys)

        if length_a == 0 and length_b == 0:
            return True

        if length_a != length_b:
            return False

        for item in item_a_keys:
            if item not in item_b_keys:
                return False
        return True

    try:
        data = json.loads(re.sub(r'(:\s*\[?\s*-?\d*)\.0', r'\1.1', json_str))
        scope = data
    except Exception as e:
        return {"go": "", "error": str(e)}

    typename = format_str(typename or "AutoGenerated")
    append(f"type {typename} ")

    parse_scope(scope)

    return {"go": ''.join(go_list)}


# if __name__ == "__main__":
#     import sys

#     if len(sys.argv) > 2 and sys.argv[2] == '-big':
#         buf = []
#         for line in sys.stdin:
#             buf.append(line)
#         json_input = ''.join(buf)
#         print(json_to_go(json_input).get("go"))
#     else:
#         for line in sys.stdin:
#             json_input = line
#             print(json_to_go(json_input).get("go"))

if __name__ == "__main__":
    with open('sample.json', 'r') as file:
        json_input = file.read()
        result = json_to_go(json_input)
        print(result["go"])
