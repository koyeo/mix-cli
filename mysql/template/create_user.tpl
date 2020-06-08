{% for user in users %}
# Create User: {{ user.username }}
# ------------------------------------------------------------

DROP USER IF EXISTS `{{ user.username }}`;
CREATE USER '{{ user.username }}'@'{{ user.host }}' identified by '{{ user.password }}';
GRANT {{ user.privileges }} ON `{{ database }}`.* TO '{{ user.username }}'@'{{ user.host }}' WITH GRANT OPTION;
{% endfor %}
FLUSH PRIVILEGES;

