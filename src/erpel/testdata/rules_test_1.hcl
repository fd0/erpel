^\w{3} [ :0-9 ]{11} [._[:alnum:]-]+ asterisk\[[[:digit:]]{2,5}\]: rc_avpair_new: unknown attribute [[:digit:]]+$
# test comment
^\w{3} [ :[:digit:] ]{11} [._[:alnum:]-]+ kernel:( \[ *[[:digit:]]+\.[[:digit:]]+\ ] )? [[:space:]]+duplex mode:[[:space:]]+(full|half)$
^\w{3} [ :[:digit:] ]{11} [._[:alnum:]-]+ kernel:( \[ *[[:digit:]]+\.[[:digit:]]+\ ] )? [[:space:]]+flowctrl:[[:space:]]+a?symmetric$

                  # another comment
^\w{3} [ :[:digit:] ]{11} [._[:alnum:]-]+ kernel:( \[ *[[:digit:]]+\.[[:digit:]]+\ ] )? [[:space:]]+irq moderation:[[:space:]]+(en|dis)abled$
