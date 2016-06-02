# load ignore rules from all files in this directory
rules_dir = "/etc/erpel/rules.d"

# prefix must match at the beginning of each line
prefix = <<EOF
^\w{3} [ :0-9 ]{11} [._[:alnum:]-]+
EOF
