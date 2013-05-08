vmail
=====

vmail is a virtual mail server helper and rss to maildir feeder.

Prepare
-------
	# Requires Ubuntu 12.04 or 12.10
	$ sudo apt-get install postfix dovecot-core dovecot-imapd dovecot-sqlite sqlite3

Install
-------

Install and initialize vmail
	$ go get github.com/mb0/vmail
	$ sudo ln -s $GOPATH/bin/vmail /usr/bin

Setup the vmail user:
	$ sudo bash
	# useradd vmail
	# mkdir /home/vmail
	# chown vmail: /home/vmail
	# chsh -s /bin/false vmail
	# sudo -u vmail vmail setup

Setup postfix config files:
	# vmail config postfix_domain  > /etc/postfix/vmail_mailbox_domains.cf
	# vmail config postfix_mailbox > /etc/postfix/vmail_mailbox_maps.cf
	# vmail config postfix_alias   > /etc/postfix/vmail_alias_maps.cf
Configure postfix:
	# postconf -e 'home_mailbox = Maildir/'
	# postconf -e 'virtual_mailbox_domains = sqlite:/etc/postfix/vmail_mailbox_domains.cf'
	# postconf -e 'virtual_mailbox_maps = sqlite:/etc/postfix/vmail_mailbox_maps.cf'
	# postconf -e 'virtual_alias_maps = sqlite:/etc/postfix/vmail_alias_maps.cf'
	postconf -e 'virtual_uid_maps = static:the vmail uid'
	postconf -e 'virtual_gid_maps = static:the vmail gid'
	# postconf -e 'virtual_mailbox_base = /home/vmail'

Setup dovecot config files:
	# vmail config dovecot_auth > /etc/dovecot/conf.d/auth-vmail.conf.ext
	# vmail config dovecot_sql  > /etc/dovecot/vmail-sql.conf.ext
Configure dovecot:
	# sed --in-place 's/^!include/#!include/' /etc/dovecot/conf.d/10-auth.conf
	# echo '!include auth-vmail.conf.ext' >> /etc/dovecot/conf.d/10-auth.conf
	# sed --in-place 's/^mail_location/#mail_location/' /etc/dovecot/conf.d/10-mail.conf

Configure sasl:
	# vim /etc/dovecot/conf.d/10-master.conf
	uncomment the 'Postfix smtp-auth' block and set mode to 0660 and user, group to postfix
	# postconf -e 'smtpd_sasl_type = dovecot'
	# postconf -e 'smtpd_sasl_path = private/auth'
	# postconf -e 'smtpd_sasl_local_domain ='
	# postconf -e 'smtpd_sasl_security_options = noanonymous'
	# postconf -e 'broken_sasl_auth_clients = no'
	# postconf -e 'smtpd_sasl_auth_enable = yes'
	# postconf -e 'smtpd_recipient_restrictions = permit_sasl_authenticated,permit_mynetworks,reject_unauth_destination'

Create certificate:
	# touch vmail.key
	# chmod 600 vmail.key
	# openssl genrsa 1024 > vmail.key
	# openssl req -new -key vmail.key -x509 -days 3650 -out vmail.crt
	# openssl req -new -x509 -extensions v3_ca -keyout vmail.key.pem -out vmail.crt.pem -days 3650
	# mv vmail.key /etc/ssl/private/
	# mv vmail.crt /etc/ssl/certs/
	# mv vmail.key.pem /etc/ssl/private/
	# mv vmail.crt.pem /etc/ssl/certs/

Configure tls:
	# postconf -e 'smtp_tls_security_level = may'
	# postconf -e 'smtpd_tls_security_level = may'
	# postconf -e 'smtp_tls_note_starttls_offer = yes'
	# postconf -e 'smtpd_tls_key_file = /etc/ssl/private/vmail.key'
	# postconf -e 'smtpd_tls_cert_file = /etc/ssl/certs/vmail.crt'
	# postconf -e 'smtpd_tls_CAfile = /etc/ssl/certs/vmail.crt.pem'
	# postconf -e 'smtpd_tls_loglevel = 1'
	# postconf -e 'smtpd_tls_received_header = yes'
	# vim /etc/dovecot/conf.d/10-ssl.conf
	ssl_cert = </etc/ssl/certs/vmail.crt
	ssl_key = </etc/ssl/private/vmail.key

Configure feeds:
	# vim /etc/dovecot/conf.d/10-mail.conf
	namespace inbox {
	  type = private
	  separator = /
	  inbox = yes
	}
	namespace {
	  type = public
	  separator = /
	  prefix = Feeds/
	  location = maildir:/home/vmail/feeds:INDEX=/home/vmail/%u/feeds
	  subscriptions = no
	}
	mail_plugins = acl
	# vim /etc/dovecot/conf.d/20-imap.conf
	protocol imap {
	  mail_plugins = $mail_plugins imap_acl
	}
	# vim /etc/dovecot/conf.d/90-acl.conf
	plugin {
	  acl = vfile
	}

remember to open your firewall and restart the services.
you might also change vim /etc/dovecot/conf.d/15-lda.conf lda_mailbox_autocreate and lda_mailbox_autosubscribe

Usage
-----
	# vmail create user@host
	# vmail alias alias@other user@host
	# vmail alias client@host user@extern
	# vmail list host
	# vmail remove user@host
	# vmail feed xkcd http://xkcd.com/rss.xml
	# vmail checkfeed '*'

vmail is BSD licensed, Copyright (c) 2013 Martin Schnabel

