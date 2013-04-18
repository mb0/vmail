vmail
=====

vmail is a virtual mail server helper.

Prepare
-------
	# Requires Ubuntu 12.04
	$ sudo apt-get install postfix dovecot-core dovecot-imapd dovecot-sqlite sqlite3

Install
-------
Setup the vmail user:
	$ sudo useradd vmail
	$ sudo mkdir /home/vmail
	$ sudo chsh -s /bin/false vmail

Install and initialize vmail
	# export PATH="$PATH:$GOPATH/bin"
	$ go get github.com/mb0/vmail
	$ sudo -u vmail vmail setup

Setup postfix config files:
	$ vmail config postfix_domain  > /etc/postfix/vmail_mailbox_domains.cf
	$ vmail config postfix_mailbox > /etc/postfix/vmail_mailbox_maps.cf
	$ vmail config postfix_alias   > /etc/postfix/vmail_alias_maps.cf
Configure postfix:
	$ postconf -e 'home_mailbox = Maildir/'
	$ postconf -e 'virtual_mailbox_domains = sqlite:/etc/postfix/vmail_mailbox_domains.cf'
	$ postconf -e 'virtual_mailbox_maps = sqlite:/etc/postfix/vmail_mailbox_maps.cf'
	$ postconf -e 'virtual_alias_maps = sqlite:/etc/postfix/vmail_alias_maps.cf'
	$ postconf -e 'virtual_uid_maps = static:vmail'
	$ postconf -e 'virtual_gid_maps = static:vmail'
	$ postconf -e 'virtual_mailbox_base = /home/vmail'

Setup dovecot config files:
	$ vmail config dovecot_auth > /etc/dovecot/conf.d/auth-vmail.conf.ext
	$ vmail config dovecot_sql  > /etc/dovecot/vmail-sql.conf.ext
Configure dovecot:
	$ sed --in-place 's/^!include/#!include/' /etc/dovecot/conf.d/10-auth.conf
	$ echo '!include auth-vmail.conf.ext' >> /etc/dovecot/conf.d/10-auth.conf

Usage
-----
	$ vmail create user@host '{SHA512-CRYPT}$6$...'
	$ vmail alias alias@other user@host
	$ vmail alias client@host user@extern
	$ vmail list host

vmail is BSD licensed, Copyright (c) 2013 Martin Schnabel

