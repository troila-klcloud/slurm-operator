package config

type SSSDLdapConfig struct {
	LDAPUri                string
	LDAPSearchBase         string
	LDAPDefaultBindDN      string
	LDAPDefaultAuthtokType string
	LDAPDefaultAuthtok     string
	SkipInsecureTLS        bool
}

var LdapConfig = SSSDLdapConfig{
	LDAPUri:                "ldap://openldap.openldap.svc.cluster.local",
	LDAPSearchBase:         "DC=hpc,DC=klcloud,DC=com",
	LDAPDefaultBindDN:      "cn=admin,dc=hpc,dc=klcloud,dc=com",
	LDAPDefaultAuthtokType: "password",
	LDAPDefaultAuthtok:     "Not@SecurePassw0rd",
	SkipInsecureTLS:        true,
}
