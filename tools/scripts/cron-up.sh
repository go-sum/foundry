service cron start
sleep 2
(crontab -l 2>/dev/null; echo '0 3,8,13,18 * * * /root/.local/bin/claude -p "What is the current TAILWIND_VERSION" --output-format text >> /tmp/claude-scheduled.log 2>&1') | crontab -
crontab -
