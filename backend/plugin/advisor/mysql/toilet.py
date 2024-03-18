import os
import re

def add_code_to_file(file_path):
    lines = []
    with open(file_path, 'r', encoding='utf-8') as file:
        for line in file:
            lines.append(line)
            if re.search(r'func .*? Enter(?!Query\()', line):
                lines.append('\tif !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {\n')
                lines.append('\t\treturn\n')
                lines.append('\t}\n')
    with open(file_path, 'w', encoding='utf-8') as file:
        file.writelines(lines)

def scan_files(directory):
    for root, _, files in os.walk(directory):
        for file_name in files:
            if file_name.endswith('.go'):  # 可以根据实际需要修改文件类型
                file_path = os.path.join(root, file_name)
                add_code_to_file(file_path)

if __name__ == "__main__":
    current_directory = os.getcwd()
    scan_files(current_directory)
    print("Code injection completed successfully.")
